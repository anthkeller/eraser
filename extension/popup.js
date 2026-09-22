let currentSite = null;

async function initialize() {
  const [tab] = await chrome.tabs.query({active: true, currentWindow: true});
  try {
    const url = new URL(tab.url);
    if (url.protocol !== "http:" && url.protocol !== "https:") throw new Error("unsupported");
    currentSite = {hostname: url.hostname, origin: url.origin, addedAt: new Date().toISOString()};
    document.querySelector("#hostname").textContent = url.hostname;
    checkSupport(url.origin);
  } catch (_) {
    document.querySelector("#remember").disabled = true;
  }
}

async function checkSupport(origin) {
  const output = document.querySelector("#support");
  try {
    const response = await fetch(`${origin}/.well-known/gpc.json`, {credentials: "omit", cache: "no-store"});
    if (!response.ok) throw new Error("unknown");
    const contentType = response.headers.get("content-type") || "";
    if (!contentType.toLowerCase().includes("application/json")) throw new Error("invalid content type");
    const declaration = await response.json();
    if (declaration.gpc === true) output.textContent = `Declares GPC support${declaration.lastUpdate ? ` · ${declaration.lastUpdate}` : ""}`;
    else if (declaration.gpc === false) output.textContent = "Declares that it does not support GPC";
    else output.textContent = "No valid GPC support declaration";
  } catch (_) {
    output.textContent = "Support unknown. The preference signal is still sent.";
  }
}

document.querySelector("#remember").addEventListener("click", async () => {
  if (!currentSite) return;
  const {accountSites = []} = await chrome.storage.local.get("accountSites");
  const sites = accountSites.filter(site => site.hostname !== currentSite.hostname);
  sites.push(currentSite);
  await chrome.storage.local.set({accountSites: sites});
  document.querySelector("#message").textContent = "Saved locally for a future Eraser account workflow.";
});

initialize();
