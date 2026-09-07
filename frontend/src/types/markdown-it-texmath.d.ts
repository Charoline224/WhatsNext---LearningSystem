declare module 'markdown-it-texmath' {
  import type MarkdownIt from 'markdown-it'

  interface TexmathOptions {
    engine: unknown
    delimiters?: string | string[]
    katexOptions?: Record<string, unknown>
  }

  const texmath: MarkdownIt.PluginSimple | MarkdownIt.PluginWithOptions<TexmathOptions>
  export default texmath
}
