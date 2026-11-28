import ast
import json
import math
import os
import re
import resource
import sys

ALLOWED_AST_NODES = (
    ast.Module,
    ast.Assign,
    ast.Name,
    ast.Store,
    ast.Load,
    ast.Constant,
    ast.BinOp,
    ast.UnaryOp,
    ast.Add,
    ast.Sub,
    ast.Mult,
    ast.Div,
    ast.USub,
    ast.UAdd,
    ast.Compare,
    ast.Eq,
    ast.NotEq,
    ast.Lt,
    ast.LtE,
    ast.Gt,
    ast.GtE,
    ast.If,
    ast.For,
    ast.Pass,
    ast.Expr,
    ast.Call,
)

BLOCKED_NAMES = {
    "os", "subprocess", "socket", "sys", "open", "eval", "exec", "compile",
    "input", "breakpoint", "getattr", "setattr", "delattr", "globals",
    "locals", "vars", "dir", "type", "object", "exit", "quit", "builtins",
    "__builtins__", "__import__", "__loader__", "__spec__", "__name__",
}

IDENT_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")


def apply_limits(mem_mb):
    """Apply zero-file-write and memory caps fail-closed before reading or parsing untrusted input."""
    resource.setrlimit(resource.RLIMIT_FSIZE, (0, 0))
    mem_bytes = mem_mb * 1024 * 1024
    applied_any_mem = False
    for limit_kind in (resource.RLIMIT_AS, resource.RLIMIT_DATA):
        try:
            soft, hard = resource.getrlimit(limit_kind)
            target = mem_bytes
            if hard != resource.RLIM_INFINITY and target > hard:
                target = hard
            resource.setrlimit(limit_kind, (target, target))
            applied_any_mem = True
        except (ValueError, OSError):
            continue
    return applied_any_mem


def check_ast(source):
    try:
        tree = ast.parse(source, mode="exec")
    except SyntaxError:
        return False

    allowed_print_calls = set()
    allowed_range_calls = set()
    call_funcs = set()

    for node in ast.walk(tree):
        if not isinstance(node, ALLOWED_AST_NODES):
            return False
        if isinstance(node, ast.Expr):
            if (
                not isinstance(node.value, ast.Call)
                or not isinstance(node.value.func, ast.Name)
                or node.value.func.id != "print"
                or len(node.value.args) != 1
                or node.value.keywords
            ):
                return False
            allowed_print_calls.add(id(node.value))
            call_funcs.add(id(node.value.func))
        elif isinstance(node, ast.For):
            if not isinstance(node.target, ast.Name) or node.orelse:
                return False
            if (
                not isinstance(node.iter, ast.Call)
                or not isinstance(node.iter.func, ast.Name)
                or node.iter.func.id != "range"
                or len(node.iter.args) != 2
                or node.iter.keywords
            ):
                return False
            allowed_range_calls.add(id(node.iter))
            call_funcs.add(id(node.iter.func))
        elif isinstance(node, ast.Assign):
            if len(node.targets) != 1 or not isinstance(node.targets[0], ast.Name):
                return False
        elif isinstance(node, ast.Compare):
            if len(node.ops) != 1 or len(node.comparators) != 1:
                return False
        elif isinstance(node, ast.Constant):
            if isinstance(node.value, bool) or not isinstance(node.value, (int, float)):
                return False

    for node in ast.walk(tree):
        if isinstance(node, ast.Call):
            if id(node) not in allowed_print_calls and id(node) not in allowed_range_calls:
                return False
        elif isinstance(node, ast.Name):
            if id(node) in call_funcs:
                continue
            if node.id.startswith("_") or node.id in BLOCKED_NAMES or node.id in ("print", "range"):
                return False

    return tree


def main():
    mem_mb = int(sys.argv[1]) if len(sys.argv) > 1 else 128
    result_pipe = os.fdopen(3, "w", encoding="utf-8")

    def emit(payload):
        result_pipe.write(json.dumps(payload))
        result_pipe.flush()

    # Apply resource limits BEFORE reading stdin, parsing AST, or compiling untrusted input.
    if not apply_limits(mem_mb):
        emit({
            "status": "resource_limit",
            "variables": {},
            "error": {
                "code": "sandbox_limit_unavailable",
                "message": "sandbox could not enforce required memory limits",
            },
        })
        return

    try:
        source = sys.stdin.read()
        tree = check_ast(source)
    except MemoryError:
        emit({
            "status": "resource_limit",
            "variables": {},
            "error": {"code": "resource_limit_exceeded", "message": "program exceeded the memory limit"},
        })
        return

    if tree is False:
        emit({
            "status": "policy_violation",
            "variables": {},
            "error": {
                "code": "sandbox_policy_violation",
                "message": "disallowed Python construct rejected by sandbox policy",
            },
        })
        return

    try:
        compiled = compile(tree, "<program>", "exec")
    except MemoryError:
        emit({
            "status": "resource_limit",
            "variables": {},
            "error": {"code": "resource_limit_exceeded", "message": "program exceeded the memory limit"},
        })
        return

    _real_print = print
    _real_range = range

    def _safe_print(arg):
        _real_print(arg, flush=True)

    def _safe_range(start, stop):
        if isinstance(start, bool) or not isinstance(start, int):
            raise TypeError("for loop bounds must evaluate to integers")
        if isinstance(stop, bool) or not isinstance(stop, int):
            raise TypeError("for loop bounds must evaluate to integers")
        return _real_range(start, stop)

    safe_globals = {
        "__builtins__": {
            "print": _safe_print,
            "range": _safe_range,
        }
    }

    try:
        exec(compiled, safe_globals, safe_globals)
    except ZeroDivisionError:
        emit({
            "status": "runtime_error",
            "variables": {},
            "error": {"code": "division_by_zero", "message": "division by zero"},
        })
        return
    except NameError as exc:
        var_name = getattr(exc, "name", "")
        if isinstance(var_name, str) and IDENT_RE.match(var_name):
            msg = f"variable '{var_name}' is used before assignment"
        else:
            msg = "variable is used before assignment"
        emit({
            "status": "runtime_error",
            "variables": {},
            "error": {"code": "undefined_variable", "message": msg},
        })
        return
    except TypeError:
        emit({
            "status": "runtime_error",
            "variables": {},
            "error": {"code": "non_integer_bound", "message": "for loop bounds must evaluate to integers"},
        })
        return
    except MemoryError:
        emit({
            "status": "resource_limit",
            "variables": {},
            "error": {"code": "resource_limit_exceeded", "message": "program exceeded the memory limit"},
        })
        return
    except (OverflowError, ValueError):
        emit({
            "status": "resource_limit",
            "variables": {},
            "error": {"code": "resource_limit_exceeded", "message": "numeric value exceeded representable limits"},
        })
        return
    except Exception:
        emit({
            "status": "runtime_error",
            "variables": {},
            "error": {"code": "runtime_error", "message": "program encountered a runtime error"},
        })
        return

    variables = {}
    for k, v in safe_globals.items():
        if k == "__builtins__":
            continue
        if isinstance(v, (int, float)) and not isinstance(v, bool):
            try:
                fv = float(v)
            except OverflowError:
                emit({
                    "status": "resource_limit",
                    "variables": {},
                    "error": {"code": "resource_limit_exceeded", "message": "numeric value exceeded representable limits"},
                })
                return
            if not math.isfinite(fv):
                emit({
                    "status": "runtime_error",
                    "variables": {},
                    "error": {"code": "non_finite_result", "message": "calculation produced a non-finite number"},
                })
                return
            variables[k] = fv

    emit({"status": "succeeded", "variables": variables})


if __name__ == "__main__":
    main()
