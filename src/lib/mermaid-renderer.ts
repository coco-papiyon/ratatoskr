import mermaid from "mermaid";
import { showMermaidError } from "./viewer-format";

let initialized = false;

function initializeMermaid() {
  if (initialized) return;
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: "strict",
  });
  initialized = true;
}

export async function renderMermaidDiagrams(container: ParentNode) {
  const diagrams = Array.from(container.querySelectorAll<HTMLElement>(".mermaid-diagram"));
  if (!diagrams.length) return;

  initializeMermaid();
  await Promise.all(diagrams.map(async (diagram) => {
    try {
      await mermaid.run({ nodes: [diagram] });
    } catch {
      showMermaidError(diagram);
    }
  }));
}
