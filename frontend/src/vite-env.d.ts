/// <reference types="vite/client" />

declare module 'drawflow' {
  export default class Drawflow {
    constructor(container: HTMLElement)
    reroute: boolean
    curvature: number
    force_first_input: boolean
    editor_mode: string
    node_selected: HTMLElement | null
    zoom: number
    start(): void
    addNode(
      name: string,
      inputs: number,
      outputs: number,
      posX: number,
      posY: number,
      className: string,
      data: Record<string, unknown>,
      html: string,
      typenode?: boolean,
    ): number
    export(): Record<string, unknown>
    import(data: Record<string, unknown>): void
    clear(): void
    clearModuleSelected(): void
    zoom_in(): void
    zoom_out(): void
    zoom_reset(): void
    removeNodeId(id: string): void
    on(event: string, callback: (...args: unknown[]) => void): void
  }
}
