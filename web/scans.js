(() => {
  "use strict";

  let api;
  let toast;
  let openProfile;
  let scans = [];
  let goals = {};
  let profile = null;
  let chart;
  let initialized = false;

  const metrics = {
    weight: { label: "Weight", unit: "kg", color: "#17201a" },
    body_fat: { label: "Body fat", unit: "%", color: "#8aa54b" },
    smm: { label: "Skeletal muscle", unit: "kg", color: "#758a7a" }
  };

  const escapeHTML = value => String(value ?? "").replace(/[&<>'"]/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[char]);
  const formatValue = (value, key) => value == null ? "—" : `${Number(value).toFixed(1)} <small>${metrics[key].unit}</small>`;
  const displayDate = value => new Intl.DateTimeFormat("en-MY", { day: "numeric", month: "short", year: "numeric" }).format(new Date(`${value}T00:00:00`));
  const today = () => new Date().toLocaleDateString("en-CA");

  const pageHead = (eyebrow, title, copy) => `
    <header class="page-head">
      <div><p class="eyebrow dark">${eyebrow}</p><h1>${title}</h1><p>${copy}</p></div>
      <span class="date-pill">${new Intl.DateTimeFormat("en-MY", { weekday: "short", day: "numeric", month: "long" }).format(new Date())}</span>
    </header>`;

  const latestMarkup = () => {
    const scan = scans[0];
    if (!scan) return `<article class="card latest-card"><div class="card-head"><div><h3>Latest reading</h3><p>Your baseline will appear here</p></div></div><p class="empty-note">Log your first scan to start seeing progress.</p></article>`;
    return `<article class="card latest-card">
      <div class="card-head"><div><h3>Latest reading</h3><p>Your most recent composition</p></div><span class="latest-date">${displayDate(scan.date)}</span></div>
      <div class="latest-metrics">
        ${Object.entries(metrics).map(([key, item]) => `<div class="metric-row"><span>${item.label}</span><strong>${formatValue(scan[key], key)}</strong></div>`).join("")}
      </div>
    </article>`;
  };

  const renderLog = () => {
    document.querySelector("#view-root").innerHTML = `${pageHead("Log scan", "Record today’s scan", "A few numbers now. A clearer trend over time.")}
      <div class="log-layout">
        <article class="card">
          <div class="card-head"><div><h3>Body composition</h3><p>Enter the readings shown on your scanner report.</p></div></div>
          <form id="scan-form">
            <div class="form-grid">
              <label class="span-2">Scan date<input name="date" type="date" value="${today()}" max="${today()}" required></label>
              <label>Weight<div class="unit-input"><input name="weight" type="number" min="20" max="400" step="0.1" inputmode="decimal" placeholder="75.5"><span>kg</span></div></label>
              <label>Body fat<div class="unit-input"><input name="body_fat" type="number" min="1" max="70" step="0.1" inputmode="decimal" placeholder="18.2"><span>%</span></div></label>
              <label>Skeletal muscle mass<div class="unit-input"><input name="smm" type="number" min="5" max="100" step="0.1" inputmode="decimal" placeholder="34.1"><span>kg</span></div></label>
              <label class="span-2">Notes <span class="muted">(optional)</span><textarea name="notes" maxlength="500" placeholder="Training phase, time of day, or anything worth remembering…"></textarea></label>
            </div>
            <div class="form-actions"><button class="button lime" type="submit">Save scan <span>→</span></button></div>
          </form>
        </article>
        ${latestMarkup()}
      </div>`;

    document.querySelector("#scan-form").addEventListener("submit", createScan);
  };

  const createScan = async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector("button[type=submit]");
    button.disabled = true;
    const data = new FormData(event.currentTarget);
    const numberOrNull = name => data.get(name) === "" ? null : Number(data.get(name));
    try {
      const created = await api("/api/scans/", { method: "POST", body: JSON.stringify({
        date: data.get("date"), weight: numberOrNull("weight"), body_fat: numberOrNull("body_fat"),
        smm: numberOrNull("smm"), notes: data.get("notes") || null
      }) });
      scans = [created, ...scans.filter(scan => scan.id !== created.id)].sort((a, b) => b.date.localeCompare(a.date));
      toast("Scan saved");
      renderLog();
    } catch (error) { toast(error.message, "error"); button.disabled = false; }
  };

  const change = (key) => {
    if (scans.length < 2 || scans[0][key] == null || scans[1][key] == null) return null;
    return Number(scans[0][key]) - Number(scans[1][key]);
  };

  const summaryCard = key => {
    const value = scans[0]?.[key];
    const delta = change(key);
    const direction = delta == null || delta === 0 ? "neutral" : delta > 0 ? "up" : "down";
    const arrow = delta == null || delta === 0 ? "—" : delta > 0 ? "↑" : "↓";
    return `<article class="card summary-card"><div><span class="label">${metrics[key].label}</span><div class="summary-value"><strong>${value == null ? "—" : Number(value).toFixed(1)}</strong><small>${value == null ? "" : metrics[key].unit}</small></div></div><span class="delta ${direction}">${arrow} ${delta == null ? "No prior" : Math.abs(delta).toFixed(1)}</span></article>`;
  };

  const renderTrends = () => {
    document.querySelector("#view-root").innerHTML = `${pageHead("Trends", "See the bigger picture", "Daily readings fluctuate. Direction is what matters.")}
      <div class="summary-grid">${Object.keys(metrics).map(summaryCard).join("")}</div>
      <article class="card chart-card">
        <div class="card-head"><div><h3>Progress over time</h3><p>${scans.length} scan${scans.length === 1 ? "" : "s"} recorded</p></div><div class="chart-toolbar">${Object.entries(metrics).map(([key, item], index) => `<button class="chart-toggle ${index === 0 ? "active" : ""}" data-metric="${key}">${item.label}</button>`).join("")}</div></div>
        <div class="chart-wrap">${scans.length ? '<canvas id="trend-chart"></canvas>' : '<p class="empty-note">Your chart will appear after you log a scan.</p>'}</div>
      </article>
      <article class="card history-card"><div class="card-head"><div><h3>Scan history</h3><p>Most recent first</p></div></div>${historyMarkup()}</article>`;
    if (scans.length) drawChart("weight");
    document.querySelectorAll(".chart-toggle").forEach(button => button.addEventListener("click", () => {
      document.querySelectorAll(".chart-toggle").forEach(item => item.classList.toggle("active", item === button));
      drawChart(button.dataset.metric);
    }));
    document.querySelectorAll(".delete-scan").forEach(button => button.addEventListener("click", () => deleteScan(button.dataset.id)));
  };

  const historyMarkup = () => {
    if (!scans.length) return '<p class="empty-note">No scans yet. Add your first one from Log scan.</p>';
    return `<div class="history-row header"><span>Date</span><span>Weight</span><span>Body fat</span><span>SMM</span><span></span></div><ul class="history-list">${scans.map(scan => `<li class="history-row"><strong>${displayDate(scan.date)}</strong><span>${scan.weight == null ? "—" : `${Number(scan.weight).toFixed(1)} kg`}</span><span>${scan.body_fat == null ? "—" : `${Number(scan.body_fat).toFixed(1)}%`}</span><span>${scan.smm == null ? "—" : `${Number(scan.smm).toFixed(1)} kg`}</span><button class="delete-scan" data-id="${escapeHTML(scan.id)}" aria-label="Delete scan from ${displayDate(scan.date)}">×</button></li>`).join("")}</ul>`;
  };

  const drawChart = key => {
    const canvas = document.querySelector("#trend-chart");
    if (!canvas || !window.Chart) return;
    chart?.destroy();
    const ordered = [...scans].reverse();
    chart = new Chart(canvas, {
      type: "line",
      data: { labels: ordered.map(scan => displayDate(scan.date)), datasets: [{ data: ordered.map(scan => scan[key]), borderColor: metrics[key].color, backgroundColor: `${metrics[key].color}18`, fill: true, borderWidth: 2.5, pointRadius: ordered.length > 14 ? 0 : 3, pointHoverRadius: 5, tension: .35, spanGaps: true }] },
      options: { responsive: true, maintainAspectRatio: false, interaction: { intersect: false, mode: "index" }, plugins: { legend: { display: false }, tooltip: { callbacks: { label: item => `${metrics[key].label}: ${item.formattedValue} ${metrics[key].unit}` } } }, scales: { x: { grid: { display: false }, ticks: { color: "#89938c", maxTicksLimit: 7, font: { size: 10 } } }, y: { border: { display: false }, grid: { color: "#edf0ed" }, ticks: { color: "#89938c", font: { size: 10 }, callback: value => `${value}${metrics[key].unit === "%" ? "%" : ""}` } } } }
    });
  };

  const deleteScan = async id => {
    if (!window.confirm("Delete this scan? This cannot be undone.")) return;
    try {
      await api(`/api/scans/${encodeURIComponent(id)}`, { method: "DELETE" });
      scans = scans.filter(scan => scan.id !== id);
      toast("Scan deleted");
      renderTrends();
    } catch (error) { toast(error.message, "error"); }
  };

  const goalProgress = key => {
    const current = scans[0]?.[key];
    const target = goals[key];
    if (current == null || target == null) return `<div class="goal-progress"><div class="goal-progress-head"><span>${metrics[key].label}</span><strong>Set a goal to track progress</strong></div><div class="progress-track"><i style="width:0"></i></div></div>`;
    const prior = scans.at(-1)?.[key];
    let percent = 50;
    if (prior != null && Number(prior) !== Number(target)) percent = Math.max(0, Math.min(100, Math.abs((Number(prior) - Number(current)) / (Number(prior) - Number(target))) * 100));
    return `<div class="goal-progress"><div class="goal-progress-head"><span>${metrics[key].label}</span><strong>${Number(current).toFixed(1)} → ${Number(target).toFixed(1)} ${metrics[key].unit}</strong></div><div class="progress-track"><i style="width:${percent}%"></i></div><div class="goal-current"><span>Starting point</span><span>${Math.round(percent)}% toward target</span></div></div>`;
  };

  const renderGoals = () => {
    document.querySelector("#view-root").innerHTML = `${pageHead("Goals", "Set your direction", "Clear targets turn individual scans into a useful journey.")}
      <div class="goals-layout">
        <article class="card"><div class="card-head"><div><h3>Target metrics</h3><p>Leave any metric blank if you’re not targeting it.</p></div></div>
          <form id="goals-form" class="goals-form">
            <label>Target weight<div class="unit-input"><input name="weight" type="number" min="20" max="400" step="0.1" value="${goals.weight ?? ""}" placeholder="70.0"><span>kg</span></div><small id="weight-estimate-note" class="field-note"></small></label>
            <label>Target body fat<div class="unit-input"><input name="body_fat" type="number" min="1" max="70" step="0.1" value="${goals.body_fat ?? ""}" placeholder="15.0"><span>%</span></div><small class="field-note">Entering a target estimates your weight while keeping current lean mass.</small></label>
            <label>Target skeletal muscle<div class="unit-input"><input name="smm" type="number" min="5" max="100" step="0.1" value="${goals.smm ?? ""}" placeholder="35.0"><span>kg</span></div></label>
            <button class="button primary" type="submit">Save goals</button>
          </form>
        </article>
        <article class="card"><div class="card-head"><div><h3>Goal progress</h3><p>Based on your first and latest readings</p></div></div><div class="goal-progress-list">${Object.keys(metrics).map(goalProgress).join("")}</div></article>
      </div>`;
    document.querySelector("#goals-form").addEventListener("submit", saveGoals);
    setupGoalEstimate();
  };

  const setupGoalEstimate = () => {
    const form = document.querySelector("#goals-form");
    const weightInput = form.elements.weight;
    const bodyFatInput = form.elements.body_fat;
    const note = document.querySelector("#weight-estimate-note");
    const latest = scans[0];
    let weightWasEdited = false;

    if (latest?.weight == null || latest?.body_fat == null) {
      note.textContent = "Log weight and body fat together to enable estimates.";
      return;
    }

    weightInput.addEventListener("input", event => {
      if (event.isTrusted) {
        weightWasEdited = true;
        note.textContent = "Manually set";
      }
    });

    bodyFatInput.addEventListener("input", () => {
      const targetBodyFat = Number(bodyFatInput.value);
      if (!bodyFatInput.value || targetBodyFat <= 0 || targetBodyFat >= 100) return;
      if (weightWasEdited && weightInput.value !== "") return;

      const leanMass = Number(latest.weight) * (1 - Number(latest.body_fat) / 100);
      const estimatedWeight = leanMass / (1 - targetBodyFat / 100);
      weightInput.value = estimatedWeight.toFixed(1);
      note.textContent = `Estimated from ${leanMass.toFixed(1)} kg current lean mass`;
    });
  };

  const saveGoals = async event => {
    event.preventDefault();
    const button = event.currentTarget.querySelector("button");
    button.disabled = true;
    const data = new FormData(event.currentTarget);
    const value = key => data.get(key) === "" ? null : Number(data.get(key));
    try {
      goals = await api("/api/goals/", { method: "PUT", body: JSON.stringify({ weight: value("weight"), body_fat: value("body_fat"), smm: value("smm") }) });
      toast("Goals saved");
      renderGoals();
    } catch (error) { toast(error.message, "error"); button.disabled = false; }
  };

  const renderRoute = () => {
    const view = ["log", "trends", "goals"].includes(location.hash.slice(1)) ? location.hash.slice(1) : "log";
    document.querySelectorAll(".nav-item[data-view]").forEach(link => link.classList.toggle("active", link.dataset.view === view));
    chart?.destroy(); chart = null;
    ({ log: renderLog, trends: renderTrends, goals: renderGoals })[view]();
  };

  const load = async () => {
    const [scanData, goalData, profileData] = await Promise.all([api("/api/scans/"), api("/api/goals/"), api("/api/profile/")]);
    scans = scanData || [];
    goals = goalData || {};
    profile = profileData;
  };

  window.CompoScans = {
    async init(dependencies) {
      api = dependencies.api; toast = dependencies.toast; openProfile = dependencies.openProfile;
      try {
        await load();
        renderRoute();
        if (!initialized) window.addEventListener("hashchange", renderRoute);
        initialized = true;
        if (!profile) openProfile(true);
      } catch (error) { toast(error.message, "error"); }
    },
    reset() { scans = []; goals = {}; profile = null; chart?.destroy(); chart = null; },
    getProfile() { return profile; },
    async saveProfile(value) { profile = await api("/api/profile/", { method: "PUT", body: JSON.stringify(value) }); return profile; }
  };
})();
