---
title: "Component Operator Runtime"
---

{{< rawhtml >}}
<div class="hero-section" style="background-image: linear-gradient(rgba(0, 31, 63, 0.7), rgba(0, 61, 122, 0.7)), url('images/background.png'); background-size: cover; background-position: center; background-repeat: no-repeat; min-height: 100vh; display: flex; align-items: center; justify-content: center; text-align: center; position: relative; width: 100vw; margin-left: calc(-50vw + 50%); margin-right: calc(-50vw + 50%); padding: 20px;">
  <div class="hero-content" style="position: relative; z-index: 1;">
    <img src="images/logo.png" alt="Component Operator Runtime Logo" style="max-width: 200px; height: auto; margin: 0 auto 2rem auto; display: block;">
    <h1 style="color: #ffffff; font-size: 3.5rem; font-weight: 700; margin-bottom: 1.5rem; text-shadow: 2px 2px 4px rgba(0,0,0,0.3);">
      Component Operator Runtime
    </h1>
    
    <p style="color: #e0f2ff; font-size: 1.5rem; font-weight: 300; margin-bottom: 3rem; max-width: 700px; margin-left: auto; margin-right: auto;">
      Build Kubernetes Component Operators
    </p>
    
    <div class="hero-code-row">
      <div class="hero-code" onclick="window.location.href='docs/concepts/controller-runtime/components/'">
        <div class="hero-code-title">Reconcile Generic Components</div>
        <pre><code class="hero-go"><span class="c">// Component is the central interface that component operators have to implement.</span>
<span class="k">type</span> <span class="t">Component</span> <span class="k">interface</span> {
    client.<span class="t">Object</span>
    <span class="c">// Return a read-only accessor to the component's spec.</span>
    <span class="f">GetSpec</span>() types.<span class="t">Unstructurable</span>
    <span class="c">// Return a read-write (usually a pointer) accessor to the component's status,</span>
    <span class="f">GetStatus</span>() *<span class="t">Status</span>
}

<span class="c">// Reconciler provides the implementation of controller-runtime's Reconciler interface,</span>
<span class="c">// for a given Component type T.</span>
<span class="k">type</span> <span class="t">Reconciler</span>[<span class="t">T</span> <span class="t">Component</span>] <span class="k">struct</span> {
    <span class="c">// ...</span>
}

<span class="c">// Reconcile the given component by applying its dependent resources to the target cluster.</span>
<span class="k">func</span> (r *<span class="t">Reconciler</span>[<span class="t">T</span>]) <span class="f">Reconcile</span>(ctx context.<span class="t">Context</span>, req ctrl.<span class="t">Request</span>) (ctrl.<span class="t">Result</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}</code></pre>
      </div>
      <div class="hero-code" onclick="window.location.href='docs/concepts/controller-runtime/generators/'">
        <div class="hero-code-title">Generate Manifests</div>
        <pre><code class="hero-go"><span class="c">// Resource generator interface.</span>
<span class="k">type</span> <span class="t">Generator</span> <span class="k">interface</span> {
    <span class="c">// Generate manifests of the dependent resources.</span>
    <span class="f">Generate</span>(ctx context.<span class="t">Context</span>, namespace <span class="t">string</span>, name <span class="t">string</span>,
        parameters types.<span class="t">Unstructurable</span>) ([]client.<span class="t">Object</span>, <span class="t">error</span>)
}

<span class="c">// Create a new HelmGenerator.</span>
<span class="k">func</span> <span class="f">NewHelmGenerator</span>(fsys fs.<span class="t">FS</span>, chartPath <span class="t">string</span>) (*<span class="t">HelmGenerator</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}

<span class="c">// Create a new KustomizeGenerator.</span>
<span class="k">func</span> <span class="f">NewKustomizeGenerator</span>(fsys fs.<span class="t">FS</span>, kustomizationPath <span class="t">string</span>,
    options <span class="t">KustomizeGeneratorOptions</span>) (*<span class="t">KustomizeGenerator</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}</code></pre>
      </div>
      <div class="hero-code" onclick="window.location.href='docs/concepts/reconciler/'">
        <div class="hero-code-title">Apply and Delete Dependents</div>
        <pre><code class="hero-go"><span class="c">// The low-level Reconciler manages specified objects in the given target cluster.</span>
<span class="k">type</span> <span class="t">Reconciler</span> <span class="k">struct</span> {
    <span class="c">// ...</span>
}

<span class="c">// Apply given object manifests to the target cluster and maintain inventory.</span>
<span class="k">func</span> (r *<span class="t">Reconciler</span>) <span class="f">Apply</span>(ctx context.<span class="t">Context</span>, inventory *[]*<span class="t">InventoryItem</span>, objects []client.<span class="t">Object</span>,
    namespace <span class="t">string</span>, ownerId <span class="t">string</span>, componentDigest <span class="t">string</span>) (<span class="t">bool</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}

<span class="c">// Delete objects stored in the inventory from the target cluster and maintain inventory.</span>
<span class="k">func</span> (r *<span class="t">Reconciler</span>) <span class="f">Delete</span>(ctx context.<span class="t">Context</span>, inventory *[]*<span class="t">InventoryItem</span>,
    ownerId <span class="t">string</span>) (<span class="t">bool</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}

<span class="c">// Check if the object set defined by inventory is ready for deletion.</span>
<span class="k">func</span> (r *<span class="t">Reconciler</span>) <span class="f">IsDeletionAllowed</span>(ctx context.<span class="t">Context</span>, inventory *[]*<span class="t">InventoryItem</span>,
    ownerId <span class="t">string</span>) (<span class="t">bool</span>, <span class="t">string</span>, <span class="t">error</span>) {
    <span class="c">// ...</span>
}</code></pre>
      </div>
    </div>
    
    <div class="scroll-arrow" style="margin-top: 4rem; animation: bounce 2s infinite;">
      <a href="#features" style="text-decoration: none;">
        <i class="fa-solid fa-circle-chevron-down" style="color: #e0f2ff; font-size: 3rem;"></i>
      </a>
    </div>
  </div>
</div>

<style>
/*
 * --u is a single "unit" length that drives the whole hero code row: font size,
 * card width, gaps and paddings are all expressed as multiples of it. It scales
 * with the viewport width (capped at 7px) so the three cards shrink together and
 * stay on one line on wide screens. Once the viewport gets too narrow (see the
 * max-width: 1100px media query below) the row switches to a single, vertically
 * stacked column with a readable font instead of shrinking further.
 */
.hero-code-row {
  --u: clamp(3px, calc((100vw - 40px) / 215), 7px);
  display: flex;
  flex-wrap: wrap;
  gap: calc(var(--u) * 3.43);
  justify-content: center;
  align-items: stretch;
  max-width: 100%;
  margin: 0 auto 3rem auto;
}

.hero-go {
  font-family: 'SFMono-Regular', 'Menlo', 'Consolas', 'Courier New', monospace;
  font-size: var(--u);
  line-height: 1.25 !important;
  color: #c9d1d9;
  white-space: pre;
}

.hero-code pre,
.hero-code code {
  line-height: 1.25 !important;
  margin: 0 !important;
}

.hero-code pre {
  padding: calc(var(--u) * 1.667) calc(var(--u) * 2);
  overflow-x: auto;
  background: transparent;
}

.hero-code {
  width: calc(var(--u) * 66.67);
  max-width: 100%;
  box-sizing: border-box;
  text-align: left;
  background: rgba(8, 20, 38, 0.55);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border: 1px solid rgba(120, 170, 255, 0.25);
  border-radius: 12px;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
  overflow: hidden;
  transition: transform 0.3s ease, box-shadow 0.3s ease;
}

@media (hover: hover) and (pointer: fine) {
  .hero-code:hover {
    transform: scale(1.4);
    z-index: 3;
    position: relative;
    box-shadow: 0 24px 60px rgba(0, 0, 0, 0.55);
    cursor: pointer;
  }
}

.hero-code-title {
  padding: calc(var(--u) * 1.143) calc(var(--u) * 2);
  background: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid rgba(120, 170, 255, 0.2);
  color: #cfe0ff;
  font-size: calc(var(--u) * 1.257);
  font-weight: 600;
  letter-spacing: 0.02em;
  font-family: 'SFMono-Regular', 'Menlo', 'Consolas', 'Courier New', monospace;
}

.hero-go .c { color: #8b949e; font-style: italic; }
.hero-go .k { color: #ff7b72; }
.hero-go .t { color: #79c0ff; }
.hero-go .f { color: #d2a8ff; }

.hero-code pre::-webkit-scrollbar { height: 8px; }
.hero-code pre::-webkit-scrollbar-thumb { background: rgba(120, 170, 255, 0.3); border-radius: 4px; }

/*
 * Below 1100px three side-by-side cards would force the shared font too small,
 * so switch the row to a single vertical column of full-width cards with a
 * comfortably readable font.
 */
@media (max-width: 1100px) {
  .hero-code-row {
    flex-direction: column;
    flex-wrap: nowrap;
    align-items: center;
    gap: 1.5rem;
    margin-bottom: 2.5rem;
    width: min(52rem, calc(100vw - 40px));
  }
  .hero-code {
    width: 100%;
    max-width: 52rem;
  }
  .hero-go { font-size: 0.8rem; }
  .hero-code pre { padding: 0.75rem 1rem; }
  .hero-code-title { font-size: 0.95rem; padding: 0.6rem 1rem; }
}

@media (max-width: 768px) {
  .hero-section {
    min-height: auto !important;
    padding: 3rem 1rem !important;
  }
  .hero-content img {
    max-width: 130px !important;
    margin-bottom: 1.5rem !important;
  }
  .hero-content h1 {
    font-size: 2rem !important;
    margin-bottom: 1rem !important;
  }
  .hero-content p {
    font-size: 1.1rem !important;
    margin-bottom: 2rem !important;
  }
  .hero-code-row {
    gap: 1rem !important;
    margin-bottom: 2rem !important;
  }
  /* Slightly smaller font on phones, where cards are already full width. */
  .hero-go { font-size: 0.75rem !important; }
  .hero-code pre { padding: 0.75rem 1rem !important; }
  .hero-code-title { font-size: 0.85rem !important; padding: 0.6rem 1rem !important; }
  .scroll-arrow { margin-top: 2rem !important; }
  .scroll-arrow i { font-size: 2.25rem !important; }
  .feature-tiles { padding: 2.5rem 1rem !important; gap: 1.25rem !important; }
  .tile { padding: 1.5rem !important; }
}

@keyframes bounce {
  0%, 20%, 50%, 80%, 100% {
    transform: translateY(0);
  }
  40% {
    transform: translateY(-10px);
  }
  60% {
    transform: translateY(-5px);
  }
}

.feature-tiles {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 2rem;
  padding: 4rem 2rem;
  max-width: 900px;
  margin: 0 auto;
  background: linear-gradient(180deg, #f8f9fa 0%, #ffffff 100%);
}

@media (max-width: 640px) {
  .feature-tiles {
    grid-template-columns: 1fr;
  }
}

.tile {
  background: #ffffff;
  border: 2px solid #003d7a;
  border-radius: 12px;
  padding: 2rem;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 6px rgba(0, 31, 63, 0.1);
  text-decoration: none;
  color: inherit;
  display: block;
}

.tile:hover {
  transform: translateY(-8px);
  box-shadow: 0 12px 20px rgba(0, 31, 63, 0.2);
  border-color: #0056b3;
}

.tile-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  color: #003d7a;
}

.tile h3 {
  color: #001f3f;
  font-size: 1.5rem;
  margin-bottom: 1rem;
  font-weight: 600;
}

.tile p {
  color: #555;
  font-size: 1rem;
  line-height: 1.6;
}
</style>

<div id="features" class="feature-tiles">
  <a href="docs/getting-started/" class="tile">
    <div class="tile-icon"><i class="fa-solid fa-rocket"></i></div>
    <h3>Getting Started</h3>
    <p>Bootstrap your first component operator in minutes, using the scaffolder or kubebuilder.</p>
  </a>

  <a href="docs/concepts/" class="tile">
    <div class="tile-icon"><i class="fa-solid fa-diagram-project"></i></div>
    <h3>Concepts</h3>
    <p>Understand components, reconcilers, generators, and how the framework fits together.</p>
  </a>

  <a href="docs/tools/" class="tile">
    <div class="tile-icon"><i class="fa-solid fa-screwdriver-wrench"></i></div>
    <h3>Tools</h3>
    <p>Discover the scaffolder and other tooling that speeds up operator development.</p>
  </a>

  <a href="docs/tutorials/" class="tile">
    <div class="tile-icon"><i class="fa-solid fa-graduation-cap"></i></div>
    <h3>Tutorials and Examples</h3>
    <p>Follow hands-on tutorials and browse practical examples to see the framework in action.</p>
  </a>
</div>
{{< /rawhtml >}}
