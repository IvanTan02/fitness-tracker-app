(() => {
  "use strict";

  const sessionKey = "compo.session";
  const authScreen = document.querySelector("#auth-screen");
  const app = document.querySelector("#app");
  const authForm = document.querySelector("#auth-form");
  let config;
  let session;
  let authMode = "signin";
  let installPrompt;

  const readSession = () => {
    try { return JSON.parse(localStorage.getItem(sessionKey)); } catch { return null; }
  };

  const saveSession = (value) => {
    session = value;
    if (value) localStorage.setItem(sessionKey, JSON.stringify(value));
    else localStorage.removeItem(sessionKey);
  };

  const toast = (message, type = "success") => {
    const el = document.createElement("div");
    el.className = `toast ${type}`;
    el.textContent = message;
    document.querySelector("#toast-region").append(el);
    window.setTimeout(() => el.remove(), 3600);
  };

  const authRequest = async (path, body) => {
    const response = await fetch(`${config.supabase_url}/auth/v1/${path}`, {
      method: "POST",
      headers: { apikey: config.supabase_publishable_key, "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(data.msg || data.error_description || data.message || "Authentication failed");
    return data;
  };

  const refreshSession = async () => {
    if (!session?.refresh_token) throw new Error("Your session has expired");
    const data = await authRequest("token?grant_type=refresh_token", { refresh_token: session.refresh_token });
    saveSession(data);
    return data.access_token;
  };

  const api = async (path, options = {}, retry = true) => {
    const headers = new Headers(options.headers || {});
    if (session?.access_token) headers.set("Authorization", `Bearer ${session.access_token}`);
    if (options.body && !(options.body instanceof FormData)) headers.set("Content-Type", "application/json");
    const response = await fetch(path, { ...options, headers });
    if (response.status === 401 && retry && session?.refresh_token) {
      await refreshSession();
      return api(path, options, false);
    }
    if (!response.ok) {
      const message = (await response.text()).trim() || `Request failed (${response.status})`;
      throw new Error(message);
    }
    if (response.status === 204) return null;
    return response.json();
  };

  const setLoading = (button, loading) => {
    button.disabled = loading;
    button.setAttribute("aria-busy", String(loading));
    const label = button.querySelector("#auth-submit-label");
    if (label) {
      label.textContent = loading ? "Working…" : authMode === "signin" ? "Sign in" : "Create account";
    } else {
      button.dataset.original ||= button.innerHTML;
      button.innerHTML = loading ? "Working…" : button.dataset.original;
    }
  };

  const showAuth = () => {
    app.hidden = true;
    authScreen.hidden = false;
  };

  const showApp = async () => {
    authScreen.hidden = true;
    app.hidden = false;
    const email = session.user?.email || "Account";
    document.querySelector("#profile-email").textContent = email;
    document.querySelector("#profile-name").textContent = email.split("@")[0];
    document.querySelectorAll("#profile-avatar, #mobile-profile").forEach(el => el.textContent = email.charAt(0).toUpperCase());
    await window.CompoScans.init({ api, toast, openProfile, user: session.user });
  };

  const openProfile = async (required = false) => {
    const dialog = document.querySelector("#profile-dialog");
    const profile = window.CompoScans.getProfile();
    document.querySelector("#profile-height").value = profile?.height_cm || "";
    document.querySelector("#profile-age").value = profile?.age || "";
    document.querySelector("#profile-gender").value = profile?.gender || "";
    dialog.dataset.required = required ? "true" : "false";
    dialog.querySelector('[value="cancel"]').hidden = required;
    document.querySelector("#sign-out").hidden = false;
    dialog.showModal();
  };

  let navScrollY = 0;

  const closeNav = () => {
    const wasScrollLocked = document.body.classList.contains("nav-open");
    document.querySelector("#sidebar").classList.remove("open");
    document.querySelector("#nav-scrim").classList.remove("open");
    document.body.classList.remove("nav-open");
    document.body.style.top = "";
    if (wasScrollLocked) window.scrollTo(0, navScrollY);
  };

  authForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = authForm.querySelector("button[type=submit]");
    setLoading(button, true);
    try {
      const credentials = {
        email: document.querySelector("#auth-email").value.trim(),
        password: document.querySelector("#auth-password").value
      };
      const data = authMode === "signin"
        ? await authRequest("token?grant_type=password", credentials)
        : await authRequest(`signup?redirect_to=${encodeURIComponent(new URL("/", window.location.origin).href)}`, credentials);
      if (!data.access_token) {
        toast("Check your inbox to confirm your email, then sign in.");
        setAuthMode("signin");
        return;
      }
      saveSession(data);
      await showApp();
    } catch (error) { toast(error.message, "error"); }
    finally { setLoading(button, false); }
  });

  const setAuthMode = (mode) => {
    authMode = mode;
    const signingIn = mode === "signin";
    document.querySelector("#auth-eyebrow").textContent = signingIn ? "Welcome back" : "Get started";
    document.querySelector("#auth-title").textContent = signingIn ? "Sign in to your account" : "Create your account";
    document.querySelector("#auth-subtitle").textContent = signingIn ? "Continue tracking your progress." : "Start building a clearer picture of your progress.";
    document.querySelector("#auth-submit-label").textContent = signingIn ? "Sign in" : "Create account";
    document.querySelector("#auth-switch-copy").textContent = signingIn ? "New to Compo?" : "Already have an account?";
    document.querySelector("#auth-switch").textContent = signingIn ? "Create an account" : "Sign in";
    document.querySelector("#auth-password").autocomplete = signingIn ? "current-password" : "new-password";
  };

  document.querySelector("#auth-switch").addEventListener("click", () => setAuthMode(authMode === "signin" ? "signup" : "signin"));
  document.querySelector("#open-nav").addEventListener("click", () => {
    navScrollY = window.scrollY;
    document.body.style.top = `-${navScrollY}px`;
    document.body.classList.add("nav-open");
    document.querySelector("#sidebar").classList.add("open");
    document.querySelector("#nav-scrim").classList.add("open");
  });
  document.querySelector("#close-nav").addEventListener("click", closeNav);
  document.querySelector("#nav-scrim").addEventListener("click", closeNav);
  document.querySelectorAll(".nav-item[data-view]").forEach(link => link.addEventListener("click", closeNav));
  document.querySelector("#profile-button").addEventListener("click", () => openProfile());
  document.querySelector("#mobile-profile").addEventListener("click", () => openProfile());
  document.querySelector("#sign-out").addEventListener("click", () => {
    saveSession(null);
    document.querySelector("#profile-dialog").close();
    window.CompoScans.reset();
    showAuth();
  });
  document.querySelector("#profile-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = event.currentTarget.querySelector('[value="save"]');
    setLoading(button, true);
    try {
      await window.CompoScans.saveProfile({
        height_cm: Number(document.querySelector("#profile-height").value),
        age: Number(document.querySelector("#profile-age").value),
        gender: document.querySelector("#profile-gender").value
      });
      document.querySelector("#profile-dialog").close();
      toast("Profile saved");
    } catch (error) { toast(error.message, "error"); }
    finally { setLoading(button, false); }
  });
  document.querySelector("#profile-dialog").addEventListener("cancel", event => {
    if (event.currentTarget.dataset.required === "true") event.preventDefault();
  });

  window.addEventListener("beforeinstallprompt", event => {
    event.preventDefault();
    installPrompt = event;
    document.querySelector("#install-app").hidden = false;
  });
  document.querySelector("#install-app").addEventListener("click", async () => {
    if (!installPrompt) return;
    await installPrompt.prompt();
    installPrompt = null;
    document.querySelector("#install-app").hidden = true;
  });
  window.addEventListener("appinstalled", () => {
    installPrompt = null;
    document.querySelector("#install-app").hidden = true;
    toast("Compo installed");
  });

  const boot = async () => {
    try {
      const response = await fetch("/api/config");
      if (!response.ok) throw new Error("App configuration is unavailable");
      config = await response.json();
      session = readSession();
      if (session?.access_token) await showApp();
      else showAuth();
    } catch (error) {
      showAuth();
      toast(error.message, "error");
    }
  };

  boot();

  if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
      navigator.serviceWorker.register("/sw.js").catch(error => {
        console.warn("Service worker registration failed:", error);
      });
    });
  }
})();
