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
    
    <div class="hero-tiles-container" style="display: flex; align-items: stretch; justify-content: center; gap: 1.5rem; max-width: 1400px; margin: 0 auto 4rem auto;">
      <div class="hero-tile" onclick="openSourcesModal()" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); border-radius: 12px; padding: 2rem 1.75rem; box-shadow: 0 8px 16px rgba(0, 31, 63, 0.15); text-align: center; border: 2px solid rgba(0, 61, 122, 0.2); transition: all 0.3s ease; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; cursor: pointer; gap: 0.6rem;">
        <i class="fa-solid fa-puzzle-piece" style="color: #003d7a; font-size: 2rem; margin-bottom: 0.25rem;"></i>
        <h3 style="color: #001f3f; font-size: 1.3rem; font-weight: 700; margin: 0;">Start Compatible</h3>
        <p style="color: #001f3f; font-size: 1rem; font-weight: 400; line-height: 1.6; margin: 0;">Use existing Helm Charts and Kustomizations</p>
      </div>
      
      <i class="fa-solid fa-circle-chevron-right" style="color: #e0f2ff; font-size: 2.5rem; flex-shrink: 0; align-self: center;"></i>
      
      <div class="hero-tile" onclick="openTemplateModal()" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); border-radius: 12px; padding: 2rem 1.75rem; box-shadow: 0 8px 16px rgba(0, 31, 63, 0.15); text-align: center; border: 2px solid rgba(0, 61, 122, 0.2); transition: all 0.3s ease; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; cursor: pointer; gap: 0.6rem;">
        <i class="fa-solid fa-wand-magic-sparkles" style="color: #003d7a; font-size: 2rem; margin-bottom: 0.25rem;"></i>
        <h3 style="color: #001f3f; font-size: 1.3rem; font-weight: 700; margin: 0;">Improve Manifests</h3>
        <p style="color: #001f3f; font-size: 1rem; font-weight: 400; line-height: 1.6; margin: 0;">Boost your manifests using enhanced template syntax and reflexivity</p>
      </div>
      
      <i class="fa-solid fa-circle-chevron-right" style="color: #e0f2ff; font-size: 2.5rem; flex-shrink: 0; align-self: center;"></i>
      
      <div class="hero-tile" onclick="openGitOpsModal()" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); border-radius: 12px; padding: 2rem 1.75rem; box-shadow: 0 8px 16px rgba(0, 31, 63, 0.15); text-align: center; border: 2px solid rgba(0, 61, 122, 0.2); transition: all 0.3s ease; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; cursor: pointer; gap: 0.6rem;">
        <i class="fa-solid fa-code-branch" style="color: #003d7a; font-size: 2rem; margin-bottom: 0.25rem;"></i>
        <h3 style="color: #001f3f; font-size: 1.3rem; font-weight: 700; margin: 0;">Integrate with GitOps</h3>
        <p style="color: #001f3f; font-size: 1rem; font-weight: 400; line-height: 1.6; margin: 0;">Embrace GitOps: use FluxCD to distribute your manifests. Or use our in-cluster Blueprints</p>
      </div>
      
      <i class="fa-solid fa-circle-chevron-right" style="color: #e0f2ff; font-size: 2.5rem; flex-shrink: 0; align-self: center;"></i>
      
      <div class="hero-tile" onclick="openControlModal()" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); border-radius: 12px; padding: 2rem 1.75rem; box-shadow: 0 8px 16px rgba(0, 31, 63, 0.15); text-align: center; border: 2px solid rgba(0, 61, 122, 0.2); transition: all 0.3s ease; flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; cursor: pointer; gap: 0.6rem;">
        <i class="fa-solid fa-sliders" style="color: #003d7a; font-size: 2rem; margin-bottom: 0.25rem;"></i>
        <h3 style="color: #001f3f; font-size: 1.3rem; font-weight: 700; margin: 0;">Control Deployments</h3>
        <p style="color: #001f3f; font-size: 1rem; font-weight: 400; line-height: 1.6; margin: 0;">Have maximum control on how rendered resources are applied to Kubernetes</p>
      </div>
    </div>
    
    <div class="scroll-arrow" style="margin-top: 4rem; animation: bounce 2s infinite;">
      <a href="#features" style="text-decoration: none;">
        <i class="fa-solid fa-circle-chevron-down" style="color: #e0f2ff; font-size: 3rem;"></i>
      </a>
    </div>
  </div>
</div>

<!-- Sources Modal -->
<div id="sourcesModal" class="modal" style="display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; overflow: auto; background-color: rgba(0, 0, 0, 0.6);">
  <div class="modal-content" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); margin: 5% auto; padding: 0; border: 2px solid #003d7a; border-radius: 12px; width: 80%; max-width: 900px; box-shadow: 0 16px 32px rgba(0, 31, 63, 0.3); animation: slideDown 0.3s ease-out;">
    <div class="modal-header" style="padding: 2rem; background: linear-gradient(135deg, #001f3f 0%, #003d7a 100%); color: white; border-radius: 10px 10px 0 0; display: flex; justify-content: space-between; align-items: center;">
      <h2 style="margin: 0; font-size: 2rem;">Use Existing Helm Charts and Kustomizations</h2>
      <span class="close" onclick="closeSourcesModal()" style="color: #e0f2ff; font-size: 2.5rem; font-weight: bold; cursor: pointer; transition: color 0.3s;">&times;</span>
    </div>
    <div class="modal-body" style="padding: 2rem; max-height: 70vh; overflow-y: auto;">
      <h3 style="color: #001f3f; margin-top: 0;">Seamless Integration with FluxCD Sources</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Existing FluxCD <code>HelmRelease</code> or <code>Kustomization</code> resources can be seamlessly replaced by components. The Component Operator automatically detects if the referenced source is a Helm Chart or not, and applies the corresponding rendering logic.
      </p>
      
      <h4 style="color: #003d7a; margin-top: 2rem;">Example: Component Using Flux Sources</h4>
      <pre style="background: #f4f4f4; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #003d7a; overflow-x: auto;"><code style="color: #333; font-family: 'Courier New', monospace;">apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: app
spec:
  sourceRef:
    fluxHelmChart:
      name: app
    # gitRepository:
    #   name: app</code></pre>

      <p style="color: #333; line-height: 1.8; font-size: 1.1rem; margin-top: 2rem;">
        The Component Operator intelligently determines the source type and applies the appropriate rendering strategy, whether it's Helm templating for charts or standard YAML processing for Kustomizations.
      </p>
      <p>
      One big advantage is that one and the same resource type (<code>Component</code>) can be used with all types of sources (Helm Chart or Kustomization). This makes it much easier to declare dependencies between components.
      </p>
    </div>
  </div>
</div>

<!-- GitOps Modal -->
<div id="gitopsModal" class="modal" style="display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; overflow: auto; background-color: rgba(0, 0, 0, 0.6);">
  <div class="modal-content" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); margin: 5% auto; padding: 0; border: 2px solid #003d7a; border-radius: 12px; width: 80%; max-width: 900px; box-shadow: 0 16px 32px rgba(0, 31, 63, 0.3); animation: slideDown 0.3s ease-out;">
    <div class="modal-header" style="padding: 2rem; background: linear-gradient(135deg, #001f3f 0%, #003d7a 100%); color: white; border-radius: 10px 10px 0 0; display: flex; justify-content: space-between; align-items: center;">
      <h2 style="margin: 0; font-size: 2rem;">GitOps with Component Operator</h2>
      <span class="close" onclick="closeGitOpsModal()" style="color: #e0f2ff; font-size: 2.5rem; font-weight: bold; cursor: pointer; transition: color 0.3s;">&times;</span>
    </div>
    <div class="modal-body" style="padding: 2rem; max-height: 70vh; overflow-y: auto;">
      <h3 style="color: #001f3f; margin-top: 0;">FluxCD Integration</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Component Operator seamlessly integrates with FluxCD to enable GitOps workflows. It can be seen as a third Flux deployer (besides kustomize-controller and helm-controller), and can be used in parallel or instead of the default Flux deployers. Define your components declaratively, let Flux do the source management, and let Component Operator handle the synchronization with the Kubernetes cluster.
      </p>
      
      <h4 style="color: #003d7a; margin-top: 2rem;">Example: FluxCD Component Deployment</h4>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Use the regular Flux source types <code>GitRepository</code>, <code>HelmChart</code>, <code>OCIRepository</code> and <code>Bucket</code>.
      </p>

      <pre style="background: #f4f4f4; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #003d7a; overflow-x: auto;"><code style="color: #333; font-family: 'Courier New', monospace;"># Using a Flux GitRepository

apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: my-app
  namespace: default
spec:
  sourceRef:
    fluxGitRepository:
      name: app
  values:
    replicas: 3
    image:
      tag: latest

# Using a Flux HelmChart

apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: my-app
  namespace: default
spec:
  sourceRef:
    fluxHelmChart:
      name: app
  values:
    replicas: 3
    image:
      tag: latest</code></pre>

      <h4 style="color: #003d7a; margin-top: 2rem;">In-Cluster Blueprints</h4>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Alternatively, use our in-cluster Blueprints feature to define reusable component templates directly in your cluster.
      </p>
      
      <pre style="background: #f4f4f4; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #003d7a; overflow-x: auto;"><code style="color: #333; font-family: 'Courier New', monospace;"># Blueprint definition

apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata: 
  name: webapp
spec:
  files:
    resources.yaml: |
      # here come the templated resource manifests

# Consuming component

apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: webapp
spec:
  sourceRef:
    blueprint:
      name: webapp
    values:
      replicas: 3
      service:
        type: LoadBalancer</code></pre>
    </div>
  </div>
</div>

<!-- Control Features Modal -->
<div id="controlModal" class="modal" style="display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; overflow: auto; background-color: rgba(0, 0, 0, 0.6);">
  <div class="modal-content" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); margin: 5% auto; padding: 0; border: 2px solid #003d7a; border-radius: 12px; width: 80%; max-width: 900px; box-shadow: 0 16px 32px rgba(0, 31, 63, 0.3); animation: slideDown 0.3s ease-out;">
    <div class="modal-header" style="padding: 2rem; background: linear-gradient(135deg, #001f3f 0%, #003d7a 100%); color: white; border-radius: 10px 10px 0 0; display: flex; justify-content: space-between; align-items: center;">
      <h2 style="margin: 0; font-size: 2rem;">Maximum Control Over Resource Management</h2>
      <span class="close" onclick="closeControlModal()" style="color: #e0f2ff; font-size: 2.5rem; font-weight: bold; cursor: pointer; transition: color 0.3s;">&times;</span>
    </div>
    <div class="modal-body" style="padding: 2rem; max-height: 70vh; overflow-y: auto;">
      <h3 style="color: #001f3f; margin-top: 0;">Apply and Delete Waves</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Dependent objects can be annotated to apply or delete them in waves. This allows you to control the order in which resources are created or removed, ensuring proper dependencies are respected.
      </p>
      
      <h4 style="color: #003d7a; margin-top: 2rem;">Example: Configuring Apply and Delete Order</h4>
      <pre style="background: #f4f4f4; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #003d7a; overflow-x: auto;"><code style="color: #333; font-family: 'Courier New', monospace;">apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
  annotations:
    component-operator.cs.sap.com/apply-order: "1"
    component-operator.cs.sap.com/delete-order: "2"</code></pre>

      <h3 style="color: #001f3f; margin-top: 2.5rem;">Improved Status Detection</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        For some objects, the default status (readiness) detection is not sufficient. In these cases, the status detection can be tweaked using annotations to specify custom conditions and checks.
      </p>
      
      <h4 style="color: #003d7a; margin-top: 2rem;">Example: Custom Status Hints</h4>
      <pre style="background: #f4f4f4; padding: 1.5rem; border-radius: 8px; border-left: 4px solid #003d7a; overflow-x: auto;"><code style="color: #333; font-family: 'Courier New', monospace;">apiVersion: services.cloud.sap.com/v1
kind: ServiceInstance
metadata:
  name: destination
  annotations:
    component-operator.cs.sap.com/status-hint: hasObservedGeneration,hasReadyCondition,conditions=Succeeded</code></pre>

      <h3 style="color: #001f3f; margin-top: 2.5rem;">Smart Handling of Custom Resource Definitions</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        The Component Operator intelligently handles components containing Custom Resource Definitions. The deletion of such components is automatically postponed as long as foreign instances of the contained Custom Resource Types exist in the cluster. This keeps the operator being part of the component alive, and gives the owners of these Custom Resource Objects the chance to clean up consistently.
      </p>

      <h3 style="color: #001f3f; margin-top: 2.5rem;">Dependencies Between Components</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        As with FluxCD <code>Kustomization</code> or <code>HelmRelease</code>, dependencies between components can be defined. However other than in FluxCD, dependencies are not only honored upon applying objects, but also during deletion, in reverse order. This ensures proper cleanup sequences and maintains system consistency.
      </p>
    </div>
  </div>
</div>

<!-- Template Syntax Modal -->
<div id="templateModal" class="modal" style="display: none; position: fixed; z-index: 1000; left: 0; top: 0; width: 100%; height: 100%; overflow: auto; background-color: rgba(0, 0, 0, 0.6);">
  <div class="modal-content" style="background: linear-gradient(135deg, #ffffff 0%, #f0f8ff 100%); margin: 5% auto; padding: 0; border: 2px solid #003d7a; border-radius: 12px; width: 80%; max-width: 900px; box-shadow: 0 16px 32px rgba(0, 31, 63, 0.3); animation: slideDown 0.3s ease-out;">
    <div class="modal-header" style="padding: 2rem; background: linear-gradient(135deg, #001f3f 0%, #003d7a 100%); color: white; border-radius: 10px 10px 0 0; display: flex; justify-content: space-between; align-items: center;">
      <h2 style="margin: 0; font-size: 2rem;">Enhanced Template Syntax & Reflexivity</h2>
      <span class="close" onclick="closeTemplateModal()" style="color: #e0f2ff; font-size: 2.5rem; font-weight: bold; cursor: pointer; transition: color 0.3s;">&times;</span>
    </div>
    <div class="modal-body" style="padding: 2rem; max-height: 70vh; overflow-y: auto;">
      <h3 style="color: #001f3f; margin-top: 0;">Helm Chart Templating</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Sources which are Helm Charts use the usual Golang-based Helm template syntax. This provides the full power of Helm templating with all standard functions and capabilities.
      </p>

      <h3 style="color: #001f3f; margin-top: 2.5rem;">Enhanced Templating for Kustomizations and Plain Manifests</h3>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        Alternatively, sources can be Kustomizations or just plain YAML manifests (which is actually a special case of a Kustomization). Unlike FluxCD, these Kustomizations or plain manifests can use Golang templating as well.
      </p>
      <p style="color: #333; line-height: 1.8; font-size: 1.1rem;">
        The syntax is similar to Helm but more powerful. In particular, enhanced reflexivity functions are provided, such as:
      </p>
      
      <ul style="color: #333; line-height: 1.8; font-size: 1.1rem; margin-left: 2rem;">
        <li><code style="background: #f4f4f4; padding: 0.2rem 0.5rem; border-radius: 4px;">localLookup</code> - Look up resources in the local component context</li>
        <li><code style="background: #f4f4f4; padding: 0.2rem 0.5rem; border-radius: 4px;">lookupWithKubeConfig</code> - Look up resources with custom kubeconfig</li>
        <li><code style="background: #f4f4f4; padding: 0.2rem 0.5rem; border-radius: 4px;">lookupList</code> - Retrieve lists of resources</li>
        <li>And many more advanced reflexivity functions...</li>
      </ul>

      <p style="color: #333; line-height: 1.8; font-size: 1.1rem; margin-top: 1.5rem;">
        These enhanced functions enable powerful cross-resource references and dynamic manifest generation based on the actual state of your cluster.
      </p>
    </div>
  </div>
</div>

<style>
@keyframes slideDown {
  from {
    transform: translateY(-50px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.close:hover {
  color: #ffffff !important;
}

.hero-tile:hover {
  transform: translateY(-8px);
  box-shadow: 0 12px 24px rgba(0, 31, 63, 0.25) !important;
  border-color: rgba(0, 61, 122, 0.4) !important;
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

<script>
function openSourcesModal() {
  document.getElementById('sourcesModal').style.display = 'block';
  document.body.style.overflow = 'hidden';
}

function closeSourcesModal() {
  document.getElementById('sourcesModal').style.display = 'none';
  document.body.style.overflow = 'auto';
}

function openGitOpsModal() {
  document.getElementById('gitopsModal').style.display = 'block';
  document.body.style.overflow = 'hidden';
}

function closeGitOpsModal() {
  document.getElementById('gitopsModal').style.display = 'none';
  document.body.style.overflow = 'auto';
}

function openControlModal() {
  document.getElementById('controlModal').style.display = 'block';
  document.body.style.overflow = 'hidden';
}

function closeControlModal() {
  document.getElementById('controlModal').style.display = 'none';
  document.body.style.overflow = 'auto';
}

function openTemplateModal() {
  document.getElementById('templateModal').style.display = 'block';
  document.body.style.overflow = 'hidden';
}

function closeTemplateModal() {
  document.getElementById('templateModal').style.display = 'none';
  document.body.style.overflow = 'auto';
}

// Close modal when clicking outside of it
window.onclick = function(event) {
  var sourcesModal = document.getElementById('sourcesModal');
  var gitopsModal = document.getElementById('gitopsModal');
  var controlModal = document.getElementById('controlModal');
  var templateModal = document.getElementById('templateModal');
  if (event.target == sourcesModal) {
    closeSourcesModal();
  }
  if (event.target == gitopsModal) {
    closeGitOpsModal();
  }
  if (event.target == controlModal) {
    closeControlModal();
  }
  if (event.target == templateModal) {
    closeTemplateModal();
  }
}

// Close modal on Escape key
document.addEventListener('keydown', function(event) {
  if (event.key === 'Escape') {
    closeSourcesModal();
    closeGitOpsModal();
    closeControlModal();
    closeTemplateModal();
  }
});
</script>

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
