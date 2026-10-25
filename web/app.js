"use strict";
const API = (window.ELIAVA_API || "").replace(/\/$/, "");
const TBILISI = [41.7151, 44.8271];
const ELIAVA = [41.7271, 44.7784];
const MAX_FILES = 10, MAX_IMAGE_MB = 10, MAX_VIDEO_MB = 100;
const S = { token: localStorage.getItem("eliava_token") || "", me: null, config: {}, cats: [] };
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
const app = () => $("#app");

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}
function gel(tetri) { const v = (tetri || 0) / 100; return (Number.isInteger(v) ? v : v.toFixed(2)) + " ₾"; }
function when(d) { return new Date(d).toLocaleString(LANG === "ka" ? "ka-GE" : "en-GB", { dateStyle: "medium", timeStyle: "short" }); }
function km(n) { return t("{n} km away", { n: n.toFixed(1) }); }
function free() { return S.config.paymentsMode === "free"; }
function termsURL() { return LANG === "ka" ? "terms-ka.html" : "terms.html"; }
function flash(msg, isErr) {
  const el = document.createElement("div");
  el.className = "flash" + (isErr ? " error" : "");
  el.setAttribute("role", isErr ? "alert" : "status");
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), isErr ? 6000 : 3500);
}
async function api(method, path, body) {
  const opt = { method, headers: {} };
  if (S.token) opt.headers.Authorization = "Bearer " + S.token;
  if (body instanceof FormData) opt.body = body;
  else if (body !== undefined) { opt.headers["Content-Type"] = "application/json"; opt.body = JSON.stringify(body); }
  let res;
  try { res = await fetch(API + path, opt); }
  catch { throw new Error(t("Can't reach the server. It may be waking up, so try again in a minute.")); }
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    if (res.status === 401 && S.token) setSession("", null);
    throw new Error(tErr(data.error) || t("Request failed ({status})", { status: res.status }));
  }
  return data;
}
const mediaURL = url => /^https?:/.test(url) ? url : API + url;
function setSession(token, user) {
  S.token = token; S.me = user;
  if (token) localStorage.setItem("eliava_token", token); else localStorage.removeItem("eliava_token");
  renderNav();
}
function go(hash) { if (location.hash === hash) render(); else location.hash = hash; }
// Runs an async action from a button/form, showing errors as a toast.
async function act(fn, okMsg) {
  try { const r = await fn(); if (okMsg) flash(okMsg); return r; } catch (e) { flash(e.message, true); }
}
// Binds a handler and disables the button (or the form's buttons) until it finishes.
function bind(sel, ev, fn) {
  $$(sel).forEach(el => el.addEventListener(ev, async e => {
    if (ev === "submit") e.preventDefault();
    const btns = el.tagName === "FORM" ? $$("button", el) : el.tagName === "BUTTON" ? [el] : [];
    if (btns.some(b => b.disabled)) return;
    btns.forEach(b => { b.disabled = true; b.setAttribute("aria-busy", "true"); });
    try { await fn(e, el); } finally { btns.forEach(b => { b.disabled = false; b.removeAttribute("aria-busy"); }); }
  }));
}
function cats(kind) { return S.cats.filter(c => c.kind === kind); }
function catOptions(kind, selected, allLabel) {
  return (allLabel ? `<option value="">${esc(allLabel)}</option>` : "") +
    cats(kind).map(c => `<option value="${allLabel ? c.slug : c.id}" ${String(selected) === String(allLabel ? c.slug : c.id) ? "selected" : ""}>${esc(catName(c))}</option>`).join("");
}
function thumbs(media) {
  if (!media || !media.length) return "";
  return `<div class="thumbs">${media.map(m => m.kind === "video"
    ? `<video src="${esc(mediaURL(m.url))}" controls preload="metadata"></video>`
    : `<a href="${esc(mediaURL(m.url))}" target="_blank" rel="noopener"><img src="${esc(mediaURL(m.url))}" alt="${esc(t("Photo"))}" loading="lazy"></a>`).join("")}</div>`;
}
function emptyState(text, actions = "") {
  return `<div class="card empty"><p>${text}</p>${actions ? `<div class="actions">${actions}</div>` : ""}</div>`;
}
function joinLabel(role) {
  if (role === "technician") return free() ? t("I'm a technician: join free") : t("Join as a technician");
  return free() ? t("I'm an Eliava seller: open a shop free") : t("Open a shop");
}
function reportButton(type, id) {
  return `<button type="button" class="link" data-report="${type}:${id}">${t("Report")}</button>`;
}
function validPhone(v) {
  let d = v.replace(/[\s()+-]/g, "");
  if (!/^\d+$/.test(d)) return false;
  if (d.startsWith("00995")) d = d.slice(5); else if (d.startsWith("995") && d.length === 12) d = d.slice(3); else if (d.startsWith("0") && d.length === 10) d = d.slice(1);
  return d.length === 9 && "345".includes(d[0]);
}

/* ---------- maps ---------- */
function baseMap(el, center, zoom) {
  el.setAttribute("aria-label", t("Map"));
  const map = L.map(el).setView(center, zoom);
  L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", { maxZoom: 19, attribution: "&copy; OpenStreetMap" }).addTo(map);
  setTimeout(() => map.invalidateSize(), 50);
  return map;
}
// A map where clicking drops a pin; calls onPick({lat,lng}).
function pickerMap(el, start, onPick) {
  const map = baseMap(el, start ? [start.lat, start.lng] : TBILISI, start ? 14 : 12);
  let marker = null;
  const set = (lat, lng, pan) => {
    if (marker) marker.setLatLng([lat, lng]); else marker = L.marker([lat, lng]).addTo(map);
    if (pan) map.setView([lat, lng], 15);
    onPick({ lat, lng });
  };
  if (start) set(start.lat, start.lng);
  map.on("click", e => set(e.latlng.lat, e.latlng.lng));
  return {
    map, set,
    locate() {
      if (!navigator.geolocation) return flash(t("Your browser can't share location"), true);
      navigator.geolocation.getCurrentPosition(p => set(p.coords.latitude, p.coords.longitude, true), () => flash(t("Could not get your location. Click the map instead."), true));
    },
  };
}

/* ---------- photo/video picker with previews ---------- */
function mediaPicker(input, box) {
  let files = [];
  const draw = () => {
    box.innerHTML = files.map((f, i) => `<div class="preview">${f.type.startsWith("video/")
      ? `<video src="${f.url}" muted preload="metadata"></video>` : `<img src="${f.url}" alt="${esc(f.file.name)}">`}
      <button type="button" data-rm="${i}" aria-label="${esc(t("Remove {name}", { name: f.file.name }))}">×</button></div>`).join("");
    $$("[data-rm]", box).forEach(b => b.addEventListener("click", () => {
      const [gone] = files.splice(+b.dataset.rm, 1); URL.revokeObjectURL(gone.url); draw();
    }));
  };
  input.addEventListener("change", () => {
    for (const file of input.files) {
      const video = file.type.startsWith("video/");
      if (!video && !file.type.startsWith("image/")) { flash(t("{name}: only photos and videos are allowed", { name: file.name }), true); continue; }
      if (file.size > (video ? MAX_VIDEO_MB : MAX_IMAGE_MB) * 1048576) { flash(t("{name} is too large (max {mb} MB)", { name: file.name, mb: video ? MAX_VIDEO_MB : MAX_IMAGE_MB }), true); continue; }
      if (files.length >= MAX_FILES) { flash(t("You can upload at most {n} files", { n: MAX_FILES }), true); break; }
      files.push({ file, type: file.type, url: URL.createObjectURL(file) });
    }
    input.value = "";
    draw();
  });
  return {
    hasImage: () => files.some(f => f.type.startsWith("image/")),
    addTo(fd) { fd.delete("media"); files.forEach(f => fd.append("media", f.file)); },
  };
}

/* ---------- report dialog ---------- */
function openReport(type, id) {
  if (!S.me) { flash(t("Log in to report"), true); go("#/login"); return; }
  const d = document.createElement("dialog");
  d.innerHTML = `<form class="stack" method="dialog"><h3>${t("Report a problem")}</h3>
    <p class="muted small">${t("An admin will review it. The person you report is not told who reported them.")}</p>
    <div><label for="rr">${t("Reason")}</label><select id="rr" name="reason">${["fake", "scam", "unsafe", "spam", "other"].map(r => `<option value="${r}">${t("reason:" + r)}</option>`).join("")}</select></div>
    <div><label for="rd">${t("Details (optional)")}</label><textarea id="rd" name="details" maxlength="1000"></textarea></div>
    <div class="row"><div style="flex:0"><button type="submit">${t("Send report")}</button></div><div style="flex:0"><button type="button" class="secondary" data-cancel>${t("Cancel")}</button></div></div></form>`;
  document.body.appendChild(d);
  d.addEventListener("close", () => d.remove());
  $("[data-cancel]", d).addEventListener("click", () => d.close());
  const f = $("form", d);
  bind("dialog form", "submit", () => act(async () => {
    await api("POST", "/api/reports", { targetType: type, targetId: id, reason: f.reason.value, details: f.details.value });
    d.close();
  }, t("Thanks. An admin will review this report.")));
  d.showModal();
}

function loginFirst(roles) {
  if (!S.me) {
    app().innerHTML = `<div class="card"><h1>${t("Please log in")}</h1><p class="muted">${t("You need an account for this page.")}</p>
      <a class="btn" href="#/login">${t("Log in")}</a> <a class="btn secondary" href="#/register">${t("Create account")}</a></div>`;
    return true;
  }
  if (roles && S.me.role !== "admin" && !roles.includes(S.me.role)) {
    app().innerHTML = `<div class="card"><h1>${t("Not available")}</h1><p class="muted">${t("This page is for {roles} accounts. You are logged in as: {role}.", { roles: roles.map(roleName).join(" / "), role: roleName(S.me.role) })}</p></div>`;
    return true;
  }
  return false;
}
function registrationNotice() {
  if (!S.me || S.me.registrationActive || !["technician", "seller"].includes(S.me.role)) return "";
  return `<p class="notice">${t("Your registration isn't paid yet, so you are hidden from customers. Pay {fee} for {days} days on the {link} page.", { fee: gel(S.config.registrationFee), days: S.config.registrationDays, link: `<a href="#/account">${t("Account")}</a>` })}</p>`;
}

/* ---------- nav ---------- */
function renderNav() {
  const r = S.me?.role;
  const links = [["#/", "Home"], ["#/technicians", "Find a technician"], ["#/market", "Market"], ["#/eliava", "Eliava map"], ["#/leaderboard", "Leaderboard"]];
  if (r === "customer" || r === "admin") links.push(["#/post", "Post a problem"], ["#/my-posts", "My requests"]);
  if (r === "technician" || r === "admin") links.push(["#/jobs", "Jobs near me"]);
  if (r === "seller" || r === "admin") links.push(["#/seller", "My shop"]);
  if (r === "customer" || r === "technician") links.push(["#/orders", "My orders"]);
  if (r === "admin") links.push(["#/admin", "Admin"]);
  const cur = "#/" + location.hash.replace(/^#\/?/, "").split("?")[0];
  $("#nav").setAttribute("aria-label", t("Main menu"));
  $("#nav").innerHTML = links.map(([h, label]) => `<a href="${h}" ${h === cur ? `class="active" aria-current="page"` : ""}>${t(label)}</a>`).join("");
  const langBtn = `<button class="secondary lang" id="lang" lang="${LANG === "ka" ? "en" : "ka"}" aria-label="${LANG === "ka" ? "Switch to English" : "ქართულზე გადართვა"}">${LANG === "ka" ? "EN" : "ქარ"}</button>`;
  $("#userbox").innerHTML = S.me
    ? `<a href="#/account" style="color:#fff">${esc(S.me.name)}</a> <span class="badge">${esc(roleName(S.me.role))}</span> ${free() ? "" : `<span>${gel(S.me.balanceTetri)}</span>`} <button class="secondary" id="logout">${t("Log out")}</button> ${langBtn}`
    : `<a class="btn secondary" href="#/login">${t("Log in")}</a> <a class="btn" href="#/register">${t("Sign up")}</a> ${langBtn}`;
  $("#logout")?.addEventListener("click", async () => { await act(() => api("POST", "/api/auth/logout")); setSession("", null); go("#/"); });
  $("#lang").addEventListener("click", () => { setLang(LANG === "ka" ? "en" : "ka"); applyStaticText(); render(); });
}
function applyStaticText() {
  document.documentElement.lang = LANG;
  $("#footer").innerHTML = `<a href="${termsURL()}">${t("Terms & Conditions")}</a> · ${t("Eliava marketplace")}`;
}
async function refreshMe() { if (S.token) { S.me = await api("GET", "/api/me").catch(() => null); renderNav(); } }

/* ---------- pages ---------- */
async function home() {
  const [tech, sell] = await Promise.all([api("GET", "/api/leaderboard?role=technician"), api("GET", "/api/leaderboard?role=seller")]);
  const top = b => b.entries.length ? `<ol>${b.entries.slice(0, 5).map(e => `<li><strong>${esc(e.name)}</strong> <span class="muted">${esc(detailText(e.detail))}</span></li>`).join("")}</ol>` : `<p class="muted">${t("Nobody yet")}</p>`;
  const boards = tech.entries.length || sell.entries.length
    ? `<div class="grid-3" style="margin-top:14px">
        <div class="card"><h2>${t("Promoted technicians")}</h2>${top(tech)}</div>
        <div class="card"><h2>${t("Promoted sellers")}</h2>${top(sell)}</div></div>`
    : `<div class="card" style="margin-top:14px"><h2>${t("Eliava is just launching")}</h2>
        <p class="muted">${t("We're signing up the first technicians and Eliava market sellers in Tbilisi. Join now and be the first customers see.")}</p>
        <div class="row"><div style="flex:0"><a class="btn" href="#/register?role=technician">${joinLabel("technician")}</a></div>
        <div style="flex:0"><a class="btn secondary" href="#/register?role=seller">${joinLabel("seller")}</a></div></div></div>`;
  app().innerHTML = `
    <div class="hero"><h1>${t("Repairs, technicians and parts, all in one place")}</h1>
      <p>${t("Find a nearby technician, post what's broken, or buy parts from the Eliava market with delivery.")}</p></div>
    <div class="grid-3">
      <div class="card service"><h2>🔧 ${t("Find a technician")}</h2><p class="muted">${t("Search by speciality and distance from you.")}</p><a class="btn" href="#/technicians">${t("Search technicians")}</a></div>
      <div class="card service"><h2>📝 ${t("Post a problem")}</h2><p class="muted">${t("Describe it, add photos/videos and your location. Get a price estimate; nearby technicians contact you.")}</p><a class="btn" href="#/post">${t("Post a problem")}</a></div>
      <div class="card service"><h2>🛒 ${t("Parts market")}</h2><p class="muted">${t("Parts and appliances from Eliava market sellers, with stall numbers on the map and home delivery.")}</p><a class="btn" href="#/market">${t("Open market")}</a></div>
    </div>${boards}`;
}

async function technicians(params) {
  let loc = null;
  app().innerHTML = `<h1>${t("Find a technician")}</h1>
    <div class="card stack">
      <div class="row">
        <div><label for="cat">${t("Speciality")}</label><select id="cat">${catOptions("repair", params.get("category") || "", t("All specialities"))}</select></div>
        <div><label for="radius">${t("Within (km)")}</label><select id="radius"><option>5</option><option selected>10</option><option>25</option><option>50</option><option>200</option></select></div>
        <div><button class="secondary" id="locate">📍 ${t("Use my location")}</button></div>
        <div><button id="search">${t("Search")}</button></div>
      </div>
      <p class="muted small">${t("Click the map to set where you need the repair.")}</p>
      <div id="map" class="map"></div>
    </div>
    <div id="results" class="grid" style="margin-top:14px" aria-live="polite"></div>`;
  let markers = [];
  const picker = pickerMap($("#map"), null, p => { loc = p; });
  $("#locate").addEventListener("click", () => picker.locate());
  const search = async () => {
    const q = new URLSearchParams({ category: $("#cat").value, radius_km: $("#radius").value });
    if (loc) { q.set("lat", loc.lat); q.set("lng", loc.lng); }
    const list = await act(() => api("GET", "/api/technicians?" + q));
    if (!list) return;
    markers.forEach(m => m.remove());
    markers = list.filter(x => x.location).map(x => L.circleMarker([x.location.lat, x.location.lng], { radius: 8, color: "#0e7c66" }).addTo(picker.map).bindPopup(esc(x.name)));
    $("#results").innerHTML = list.length ? list.map(x => `
      <div class="card">
        <h2>${esc(x.name)} ${x.boostTetri > 0 ? `<span class="badge top">${t("Promoted")}</span>` : ""}</h2>
        <div>${x.categories.map(c => `<span class="badge">${esc(catName(c))}</span>`).join("")}</div>
        <p class="muted">${esc(x.city)}${x.distanceKm != null ? ` · ${km(x.distanceKm)}` : ""} · ${t("works within {n} km", { n: x.serviceRadiusKm })}</p>
        <p>${esc(x.bio)}</p>
        <a class="btn" href="tel:${esc(x.phone)}">📞 ${esc(x.phone)}</a>
        ${S.me?.id === x.id ? "" : `<p>${reportButton("user", x.id)}</p>`}
      </div>`).join("") : emptyState(t("No technicians found here yet. Eliava is just launching. Post your problem and nearby technicians will see it as they join."),
        `<a class="btn" href="#/post">${t("Post a problem")}</a><a class="btn secondary" href="#/register?role=technician">${joinLabel("technician")}</a>`);
  };
  bind("#search", "click", search);
  search();
}

async function newPost() {
  if (loginFirst(["customer"])) return;
  let loc = null;
  app().innerHTML = `<h1>${t("Post a problem")}</h1>
    <form class="card stack" id="f">
      <div class="row">
        <div><label for="pc">${t("What needs fixing?")}</label><select id="pc" name="categoryId" required>${catOptions("repair")}</select></div>
        <div><label for="pt">${t("Short title")}</label><input id="pt" name="title" required minlength="5" maxlength="120" placeholder="${esc(t("e.g. AC blows warm air"))}"></div>
      </div>
      <div><label for="pd">${t("Description (at least 20 characters)")}</label><textarea id="pd" name="description" required minlength="20" placeholder="${esc(t("Brand, model, what happens, since when…"))}"></textarea></div>
      <label class="check"><input type="checkbox" name="hasParts" value="true"> ${t("I already have the needed parts at home")}</label>
      <div><label for="pm">${t("Photos and videos (at least one photo; photos up to 10 MB, videos up to 100 MB)")}</label><input id="pm" type="file" multiple accept="image/*,video/*">
        <div class="previews" id="previews"></div></div>
      <div><label for="pa">${t("Address (optional; technicians only see the area until they view your number)")}</label><input id="pa" name="address" maxlength="300" autocomplete="street-address"></div>
      <div class="row"><div><label>${t("Location: click the map")}</label></div><div style="flex:0"><button type="button" class="secondary" id="locate">📍 ${t("Use my location")}</button></div></div>
      <div id="map" class="map"></div>
      <div id="estimate" aria-live="polite"></div>
      <div class="row"><div style="flex:0"><button type="button" class="secondary" id="est">${t("Get price estimate")}</button></div><div style="flex:0"><button type="submit">${t("Post request")}</button></div></div>
    </form>`;
  const picker = pickerMap($("#map"), null, p => { loc = p; });
  $("#locate").addEventListener("click", () => picker.locate());
  const media = mediaPicker($("#pm"), $("#previews"));
  const f = $("#f");
  const selectedCat = () => S.cats.find(c => c.id === +f.categoryId.value);
  const showEstimate = e => {
    $("#estimate").innerHTML = `<div class="estimate"><strong>${t("Estimated cost: {min} – {max}", { min: gel(e.minTetri), max: gel(e.maxTetri) })}</strong><br><span class="muted">${esc(estimateNote(e, selectedCat()))}${e.source === "ai" ? ` (${t("AI estimate")})` : ""}</span></div>`;
  };
  bind("#est", "click", () => act(async () => {
    if (!loc) throw new Error(t("Pick your location on the map first"));
    showEstimate(await api("POST", "/api/estimate", { categoryId: +f.categoryId.value, description: f.description.value, hasParts: f.hasParts.checked, location: loc }));
  }));
  bind("#f", "submit", () => act(async () => {
    if (!loc) throw new Error(t("Pick your location on the map"));
    if (!media.hasImage()) throw new Error(t("Add at least one photo of the problem"));
    const fd = new FormData(f);
    media.addTo(fd);
    fd.set("lat", loc.lat); fd.set("lng", loc.lng); fd.set("hasParts", f.hasParts.checked ? "true" : "false");
    const btn = $("button[type=submit]", f); btn.textContent = t("Uploading…");
    try {
      const p = await api("POST", "/api/posts", fd);
      if (p.flagged) flash(t("Posted, but it looks like a repeat of one of your earlier requests, so it is hidden from technicians until an admin reviews it."), true);
      else flash(t("Your request is live! Estimated {min} – {max}", { min: gel(p.estimate.minTetri), max: gel(p.estimate.maxTetri) }));
      go("#/my-posts");
    } finally { btn.textContent = t("Post request"); }
  }));
}

function postCard(p, mode) {
  const status = p.status === "open" ? `<span class="badge ok">${t("open")}</span>` : `<span class="badge">${t("closed")}</span>`;
  const flagged = p.flagged ? `<span class="badge warn">${t("flagged")}: ${esc(p.flagReason)}</span>` : "";
  let actions = "";
  if (mode === "owner") {
    actions = `<p class="muted">${t("{n} technician(s) viewed your number", { n: p.viewCount })}</p>
      <button class="secondary" data-viewers="${p.id}">${t("Who viewed?")}</button>
      ${p.status === "open" ? `<button class="secondary" data-close="${p.id}">${t("Mark solved / close")}</button>` : ""}
      <div id="viewers-${p.id}"></div>`;
  } else if (mode === "tech") {
    actions = (p.contact
      ? `<p><strong>${esc(p.contact.name)}</strong> · <a href="tel:${esc(p.contact.phone)}">📞 ${esc(p.contact.phone)}</a>${p.address ? `<br><span class="muted">${esc(p.address)}</span>` : ""}</p>`
      : `<button class="accent" data-reveal="${p.id}">${S.config.revealPrice ? t("View number ({price})", { price: gel(S.config.revealPrice) }) : t("View number (free)")}</button>`)
      + `<p>${reportButton("post", p.id)}</p>`;
  }
  return `<div class="card">
    <h2>${esc(p.title)}</h2>
    <div>${status}<span class="badge">${esc(catName(p.category))}</span>${p.hasParts ? `<span class="badge">${t("has parts")}</span>` : ""}${flagged}</div>
    <p>${esc(p.description)}</p>
    ${thumbs(p.media)}
    <p class="muted">${t("Estimate {min} – {max}", { min: gel(p.estimate.minTetri), max: gel(p.estimate.maxTetri) })}${p.distanceKm != null ? ` · ${km(p.distanceKm)}` : ""} · ${when(p.createdAt)}</p>
    ${actions}
  </div>`;
}

async function myPosts() {
  if (loginFirst(["customer"])) return;
  const posts = await api("GET", "/api/posts/mine");
  app().innerHTML = `<h1>${t("My repair requests")}</h1><p><a class="btn" href="#/post">+ ${t("New request")}</a></p>
    <div class="grid">${posts.map(p => postCard(p, "owner")).join("") || emptyState(t("No requests yet. Post your first problem and nearby technicians will see it."), `<a class="btn" href="#/post">${t("Post a problem")}</a>`)}</div>`;
  bind("[data-viewers]", "click", (e, el) => act(async () => {
    const v = await api("GET", `/api/posts/${el.dataset.viewers}/viewers`);
    $("#viewers-" + el.dataset.viewers).innerHTML = v.length ? `<ul>${v.map(x => `<li>${esc(x.name)} · ${when(x.viewedAt)}</li>`).join("")}</ul>` : `<p class="muted">${t("Nobody yet")}</p>`;
  }));
  bind("[data-close]", "click", (e, el) => act(async () => { await api("POST", `/api/posts/${el.dataset.close}/close`); myPosts(); }, t("Request closed")));
}

async function jobs() {
  if (loginFirst(["technician"])) return;
  app().innerHTML = `<h1>${t("Jobs near me")}</h1>${registrationNotice()}
    <div class="card row">
      <div><label for="cat">${t("Category")}</label><select id="cat">${catOptions("repair", "", t("All categories"))}</select></div>
      <div><label for="radius">${t("Within (km, empty = your service radius)")}</label><input id="radius" type="number" min="1" max="500" inputmode="numeric"></div>
      <div><label class="check"><input type="checkbox" id="mine" checked> ${t("Only my specialities")}</label></div>
      <div><button id="load">${t("Refresh")}</button></div>
    </div>
    <div id="list" class="grid" style="margin-top:14px" aria-live="polite"></div>`;
  const load = async () => {
    const q = new URLSearchParams({ category: $("#cat").value, mine: $("#mine").checked ? "1" : "0" });
    if ($("#radius").value) q.set("radius_km", $("#radius").value);
    const posts = await act(() => api("GET", "/api/posts/nearby?" + q));
    if (!posts) { $("#list").innerHTML = emptyState(t("Set your location and specialities on your Account page first."), `<a class="btn" href="#/account">${t("Account")}</a>`); return; }
    $("#list").innerHTML = posts.map(p => postCard(p, "tech")).join("") || emptyState(t("No open requests nearby right now. Try a bigger distance or untick “Only my specialities”."));
    bind("[data-reveal]", "click", (e, el) => act(async () => {
      const msg = S.config.revealPrice ? t("View this customer's number for {price}? You are charged once per request.", { price: gel(S.config.revealPrice) }) : t("View this customer's number? The customer will see that you viewed it.");
      if (!confirm(msg)) return;
      await api("POST", `/api/posts/${el.dataset.reveal}/reveal`);
      await refreshMe(); load();
    }));
  };
  bind("#load", "click", load);
  load();
}

function listingCard(l) {
  const img = l.media.find(m => m.kind === "image");
  const stall = l.seller.stallCode ? `<a href="#/eliava?stall=${l.seller.stallId}" class="badge">${t("Eliava stall {code}", { code: esc(l.seller.stallCode) })}</a>` : "";
  return `<div class="card">
    ${img ? `<img class="cover" src="${esc(mediaURL(img.url))}" alt="${esc(l.title)}" loading="lazy">` : `<div class="cover"></div>`}
    <h2>${esc(l.title)} ${l.boostTetri > 0 ? `<span class="badge top">${t("Promoted")}</span>` : ""}</h2>
    <div><span class="badge">${esc(catName(l.category))}</span>${stall}</div>
    <p class="muted">${esc(l.description)}</p>
    <p><span class="price">${gel(l.priceTetri)}</span> <span class="muted">· ${t("{n} in stock", { n: l.stock })}</span></p>
    <p class="muted small">${esc(l.seller.shopName)} · <a href="tel:${esc(l.seller.phone)}">${esc(l.seller.phone)}</a></p>
    <button data-order="${l.id}">${t("Order")}</button> ${reportButton("listing", l.id)}
    <form class="stack" id="order-${l.id}" hidden style="margin-top:10px">
      <div class="row"><div><label for="oq-${l.id}">${t("Quantity")}</label><input id="oq-${l.id}" name="quantity" type="number" min="1" max="${l.stock}" value="1" inputmode="numeric"></div></div>
      <label class="check"><input type="checkbox" name="delivery"> ${t("Deliver to my home (+{fee})", { fee: gel(S.config.deliveryFee) })}</label>
      <p class="muted small">${t("Delivery: Tbilisi only, parcels up to 20 kg. Large appliances: arrange transport with the seller. You pay the seller the item price × quantity (+ delivery fee) in cash on delivery or at pickup.")}</p>
      <div><label for="oa-${l.id}">${t("Delivery address")}</label><input id="oa-${l.id}" name="deliveryAddress" autocomplete="street-address"></div>
      <button type="submit">${t("Place order: I will pay on delivery / pickup")}</button>
    </form>
  </div>`;
}
function bindOrders(reload) {
  bind("[data-order]", "click", (e, el) => {
    if (loginFirst(["customer", "technician"])) return;
    $("#order-" + el.dataset.order).hidden = false; el.hidden = true;
  });
  bind("form[id^=order-]", "submit", (e, f) => act(async () => {
    await api("POST", "/api/orders", { listingId: +f.id.slice(6), quantity: +f.quantity.value, delivery: f.delivery.checked, deliveryAddress: f.deliveryAddress.value });
    reload();
  }, t("Order placed! The seller will contact you.")));
}

async function market(params) {
  const stalls = await api("GET", "/api/stalls");
  app().innerHTML = `<h1>${t("Parts market")}</h1>
    <div class="card row">
      <div><label for="q">${t("Search")}</label><input id="q" type="search" placeholder="${esc(t("e.g. compressor"))}" value="${esc(params.get("q") || "")}"></div>
      <div><label for="cat">${t("Category")}</label><select id="cat">${catOptions("part", params.get("category") || "", t("All categories"))}</select></div>
      <div><label for="stall">${t("Eliava stall")}</label><select id="stall"><option value="">${t("Any stall")}</option>${stalls.map(s => `<option value="${s.id}" ${params.get("stall") == s.id ? "selected" : ""}>${esc(s.code)}</option>`).join("")}</select></div>
      <div><button id="search">${t("Search")}</button></div>
    </div>
    <div id="list" class="grid" style="margin-top:14px" aria-live="polite"></div>`;
  const load = async () => {
    const q = new URLSearchParams({ q: $("#q").value, category: $("#cat").value, stall: $("#stall").value });
    const list = await act(() => api("GET", "/api/listings?" + q));
    if (!list) return;
    $("#list").innerHTML = list.map(listingCard).join("") || emptyState(
      $("#q").value || $("#cat").value || $("#stall").value ? t("Nothing found. Try another search or category.") : t("No parts listed yet. Eliava market sellers are joining now."),
      `<a class="btn secondary" href="#/register?role=seller">${joinLabel("seller")}</a>`);
    bindOrders(load);
  };
  bind("#search", "click", load);
  $("#q").addEventListener("keydown", e => { if (e.key === "Enter") load(); });
  load();
}

async function eliava(params) {
  const stalls = await api("GET", "/api/stalls");
  const isAdmin = S.me?.role === "admin";
  app().innerHTML = `<h1>${t("Eliava market map")}</h1>
    <p class="muted">${t("Click a stall to see what its sellers offer.")}${isAdmin ? ` <strong>${t("Admin")}:</strong> ${t("drag pins to their real positions; changes save automatically.")}` : ""}</p>
    <div id="map" class="map tall"></div>
    ${isAdmin ? `<form class="card row" id="add" style="margin-top:12px"><div><label for="sc">${t("New stall code")}</label><input id="sc" name="code" required placeholder="E1"></div><div><label for="sn">${t("Name")}</label><input id="sn" name="name"></div><div><button>${t("Add stall at map centre")}</button></div></form>` : ""}
    <h2 id="title"></h2><div id="list" class="grid" aria-live="polite"></div>`;
  const map = baseMap($("#map"), ELIAVA, 17);
  const showStall = async st => {
    $("#title").textContent = t("Stall {code}", { code: st.code }) + (st.name ? " · " + st.name : "");
    const list = await act(() => api("GET", "/api/listings?stall=" + st.id));
    if (!list) return;
    $("#list").innerHTML = list.map(listingCard).join("") || emptyState(t("No listings at this stall yet."), `<a class="btn secondary" href="#/register?role=seller">${joinLabel("seller")}</a>`);
    bindOrders(() => showStall(st));
  };
  stalls.forEach(st => {
    const m = L.marker([st.location.lat, st.location.lng], { draggable: isAdmin, title: t("Stall {code}", { code: st.code }) }).addTo(map)
      .bindTooltip(st.code, { permanent: true, direction: "top", offset: [-15, -12] });
    m.on("click", () => showStall(st));
    if (isAdmin) m.on("dragend", () => act(() => api("PATCH", "/api/admin/stalls/" + st.id, { location: m.getLatLng() }), t("Stall {code} moved", { code: st.code })));
  });
  const pre = stalls.find(s => String(s.id) === params.get("stall"));
  if (pre) { map.setView([pre.location.lat, pre.location.lng], 18); showStall(pre); }
  bind("#add", "submit", (e, f) => act(async () => {
    await api("POST", "/api/admin/stalls", { code: f.code.value, name: f.name.value, location: map.getCenter() });
    eliava(params);
  }, t("Stall added")));
}

async function seller() {
  if (loginFirst(["seller"])) return;
  const [profile, stalls, listings, orders] = await Promise.all([
    S.me.role === "seller" ? api("GET", "/api/me/seller-profile") : {}, api("GET", "/api/stalls"),
    api("GET", "/api/listings/mine"), api("GET", "/api/orders/incoming")]);
  let pin = profile.stallId ? null : profile.location;
  app().innerHTML = `<h1>${t("My shop")}</h1>${registrationNotice()}
    <form class="card stack" id="profile">
      <h2>${t("Shop details")}</h2>
      <div class="row">
        <div><label for="sh">${t("Shop name")}</label><input id="sh" name="shopName" required value="${esc(profile.shopName)}"></div>
        <div><label for="ss">${t("Eliava stall")}</label><select id="ss" name="stallId"><option value="">${t("Not at Eliava (pin on map below)")}</option>${stalls.map(s => `<option value="${s.id}" ${profile.stallId === s.id ? "selected" : ""}>${esc(s.code)}</option>`).join("")}</select></div>
        <div><label for="sa">${t("Address")}</label><input id="sa" name="address" value="${esc(profile.address)}"></div>
      </div>
      <p class="muted small">${t("Not at Eliava? Click the map to pin your shop.")}</p>
      <div id="map" class="map" style="height:240px"></div>
      <div><button>${t("Save shop")}</button></div>
    </form>
    <form class="card stack" id="new" style="margin-top:14px">
      <h2>${t("New listing")}</h2>
      <div class="row">
        <div><label for="lc">${t("Category")}</label><select id="lc" name="categoryId">${catOptions("part")}</select></div>
        <div><label for="lt">${t("Title")}</label><input id="lt" name="title" required minlength="3"></div>
        <div><label for="lp">${t("Price (₾)")}</label><input id="lp" name="price" required inputmode="decimal" placeholder="25.50"></div>
        <div><label for="ls">${t("In stock")}</label><input id="ls" name="stock" type="number" min="0" value="1" required inputmode="numeric"></div>
      </div>
      <div><label for="ld">${t("Description")}</label><textarea id="ld" name="description"></textarea></div>
      <div><label for="lm">${t("Photos")}</label><input id="lm" type="file" multiple accept="image/*,video/*"><div class="previews" id="lpreviews"></div></div>
      <div><button>${t("Publish listing")}</button></div>
    </form>
    <h2>${t("My listings")}</h2>
    <table><tr><th>${t("Item")}</th><th>${t("Price (₾)")}</th><th>${t("Stock")}</th><th>${t("Visible")}</th><th></th></tr>
      ${listings.map(l => `<tr data-row="${l.id}"><td>${esc(l.title)}<br><span class="muted small">${esc(catName(l.category))}</span></td>
        <td><input name="price" value="${l.priceTetri / 100}" style="width:90px" inputmode="decimal" aria-label="${esc(t("Price (₾)") + ": " + l.title)}"></td>
        <td><input name="stock" type="number" min="0" value="${l.stock}" style="width:80px" aria-label="${esc(t("Stock") + ": " + l.title)}"></td>
        <td><input name="active" type="checkbox" ${l.active ? "checked" : ""} aria-label="${esc(t("Visible") + ": " + l.title)}"></td>
        <td><button class="secondary" data-save="${l.id}">${t("Save")}</button></td></tr>`).join("") || `<tr><td colspan="5" class="muted">${t("No listings yet. Add your first one above.")}</td></tr>`}
    </table>
    <h2>${t("Incoming orders")}</h2>${ordersTable(orders, "seller")}`;
  const picker = pickerMap($("#map"), profile.location, p => { pin = p; });
  picker.map.setView(profile.location ? [profile.location.lat, profile.location.lng] : ELIAVA, 15);
  const media = mediaPicker($("#lm"), $("#lpreviews"));
  bind("#profile", "submit", (e, f) => act(async () => {
    const stallId = f.stallId.value ? +f.stallId.value : null;
    await api("PUT", "/api/me/seller-profile", { shopName: f.shopName.value, stallId, address: f.address.value, location: stallId ? null : pin });
  }, t("Shop saved")));
  bind("#new", "submit", (e, f) => act(async () => { const fd = new FormData(f); media.addTo(fd); await api("POST", "/api/listings", fd); seller(); }, t("Listing published")));
  bind("[data-save]", "click", (e, el) => act(async () => {
    const row = $(`[data-row="${el.dataset.save}"]`);
    const price = Math.round(parseFloat($("[name=price]", row).value.replace(",", ".")) * 100);
    if (!(price > 0)) throw new Error(t("Enter a valid price"));
    await api("PATCH", "/api/listings/" + el.dataset.save, { priceTetri: price, stock: +$("[name=stock]", row).value, active: $("[name=active]", row).checked });
  }, t("Saved")));
  bindOrderStatus(seller);
}

const statusLabel = s => t("status:" + s);
const sellerNext = { pending: ["confirmed", "cancelled"], confirmed: ["out_for_delivery", "completed", "cancelled"], out_for_delivery: ["completed", "cancelled"] };
function ordersTable(orders, mode) {
  if (!orders.length) return `<p class="muted">${t("No orders yet.")}</p>`;
  return `<table><tr><th>#</th><th>${t("Item")}</th><th>${mode === "seller" ? t("Customer") : t("Seller")}</th><th>${t("Qty")}</th><th>${t("Total")}</th><th>${t("Delivery")}</th><th>${t("Status")}</th></tr>
    ${orders.map(o => {
      const btns = mode === "seller" ? (sellerNext[o.status] || []) : (o.status === "pending" ? ["cancelled"] : []);
      return `<tr><td>${o.id}</td><td>${esc(o.listingTitle)}</td>
        <td>${mode === "seller" ? `${esc(o.customerName)}<br><a href="tel:${esc(o.customerPhone)}">${esc(o.customerPhone)}</a>` : `${esc(o.sellerName)}${o.stallCode ? ` (${t("Stall {code}", { code: esc(o.stallCode) })})` : ""}<br><a href="tel:${esc(o.sellerPhone)}">${esc(o.sellerPhone)}</a>`}</td>
        <td>${o.quantity}</td><td>${gel(o.totalTetri)}</td><td>${o.delivery ? esc(o.deliveryAddress) : t("Pickup")}</td>
        <td><span class="badge">${statusLabel(o.status)}</span><br>${btns.map(s => `<button class="secondary small" data-status="${s}" data-oid="${o.id}">${s === "cancelled" ? t("Cancel") : statusLabel(s)}</button>`).join(" ")}</td></tr>`;
    }).join("")}</table>`;
}
function bindOrderStatus(reload) {
  bind("[data-status]", "click", (e, el) => act(async () => { await api("PATCH", "/api/orders/" + el.dataset.oid, { status: el.dataset.status }); reload(); }, t("Order updated")));
}

async function orders() {
  if (loginFirst()) return;
  const list = await api("GET", "/api/orders/mine");
  app().innerHTML = `<h1>${t("My orders")}</h1>${list.length ? ordersTable(list, "buyer") : emptyState(t("No orders yet."), `<a class="btn" href="#/market">${t("Open market")}</a>`)}`;
  bindOrderStatus(orders);
}

async function account() {
  if (loginFirst()) return;
  const w = await api("GET", "/api/wallet");
  S.me = w.user; renderNav();
  const u = S.me, pro = ["technician", "seller"].includes(u.role);
  const board = pro && !free() ? await api("GET", "/api/leaderboard?role=" + u.role) : null;
  const tp = u.role === "technician" ? await api("GET", "/api/me/technician-profile") : null;
  const topAmount = board?.entries[0]?.boostTetri || 0;
  let loc = tp?.location || null;
  app().innerHTML = `<h1>${t("My account")}</h1>
    <div class="grid-3">
      <div class="card"><h2>${esc(u.name)}</h2><p class="muted">${esc(u.email)} · ${esc(u.phone)}<br>${t("Account type")}: ${esc(roleName(u.role))}</p></div>
      <div class="card"><h2>${free() ? t("Payments") : t("Wallet: {amount}", { amount: gel(u.balanceTetri) })}</h2>
        ${S.config.paymentsMode === "mock" ? `<form class="row" id="topup"><div><label for="ta">${t("Top up (₾)")}</label><input id="ta" name="amount" type="number" min="1" max="1000" value="20"></div><div><button>${t("Top up")}</button></div></form>
        <p class="muted small">${t("Test mode: no real money is charged.")}</p>` : free() ? `<p class="muted small">${t("Free beta: registration and viewing numbers are free. No payments are taken.")}</p>` : ""}</div>
      ${pro ? `<div class="card"><h2>${t("Registration")}</h2>
        <p>${u.registrationActive ? `<span class="badge ok">${t("Active")}</span> ${free() ? t("free during the beta") : t("until {date}", { date: when(u.registrationPaidUntil) })}` : `<span class="badge warn">${t("Not paid")}</span> ${t("you're hidden from customers")}`}</p>
        ${free() ? "" : `<button id="reg">${t("Pay {fee} for {days} days", { fee: gel(S.config.registrationFee), days: S.config.registrationDays })}</button>`}</div>` : ""}
    </div>
    ${pro && !free() ? `<div class="card" style="margin-top:14px"><h2>${t("Leaderboard boost")}</h2>
      <p class="muted">${t("Your active boost: {mine}. Current #1 has {top}. Pay more than that to get to the top for {days} days. Boosted profiles are labelled “Promoted”.", { mine: gel(w.activeBoostTetri), top: gel(topAmount), days: S.config.boostDays })}</p>
      <form class="row" id="boost"><div><label for="ba">${t("Boost amount (₾)")}</label><input id="ba" name="amount" type="number" min="1" value="${Math.max(5, Math.floor((topAmount - w.activeBoostTetri) / 100) + 1)}"></div><div><button class="accent">${t("Boost")}</button></div></form></div>` : ""}
    ${tp ? `<form class="card stack" id="tp" style="margin-top:14px"><h2>${t("Technician profile")}</h2>
      <div class="row"><div><label for="tc">${t("City")}</label><input id="tc" name="city" value="${esc(tp.city)}"></div>
        <div><label for="tr">${t("I travel up to (km)")}</label><input id="tr" name="radius" type="number" min="1" max="500" value="${tp.serviceRadiusKm}" inputmode="numeric"></div></div>
      <div><label for="tb">${t("About me")}</label><textarea id="tb" name="bio">${esc(tp.bio)}</textarea></div>
      <fieldset style="border:0;padding:0;margin:0"><legend class="muted" style="font-weight:600;font-size:13px">${t("Specialities")}</legend>${cats("repair").map(c => `<label class="check"><input type="checkbox" name="cat" value="${c.id}" ${tp.categories.some(x => x.id === c.id) ? "checked" : ""}> ${esc(catName(c))}</label>`).join("")}</fieldset>
      <div class="row"><div><label>${t("My base location: click the map")}</label></div><div style="flex:0"><button type="button" class="secondary" id="locate">📍 ${t("Use my location")}</button></div></div>
      <div id="map" class="map"></div>
      <div><button>${t("Save profile")}</button></div></form>` : ""}
    ${w.transactions.length || !free() ? `<h2>${t("Wallet history")}</h2>
    <table><tr><th>${t("When")}</th><th>${t("What")}</th><th>${t("Amount")}</th></tr>${w.transactions.map(x => `<tr><td>${when(x.createdAt)}</td><td>${esc(reasonText(x.reason))}</td><td>${x.amountTetri > 0 ? "+" : ""}${gel(x.amountTetri)}</td></tr>`).join("") || `<tr><td colspan="3" class="muted">${t("Nothing yet.")}</td></tr>`}</table>` : ""}`;
  bind("#topup", "submit", (e, f) => act(async () => { await api("POST", "/api/wallet/topup", { amountTetri: Math.round(+f.amount.value * 100) }); account(); }, t("Wallet topped up")));
  bind("#reg", "click", () => act(async () => { await api("POST", "/api/registration/pay"); account(); }, t("Registration paid")));
  bind("#boost", "submit", (e, f) => act(async () => { await api("POST", "/api/boosts", { amountTetri: Math.round(+f.amount.value * 100) }); account(); }, t("Boost active!")));
  if (u.role !== "admin") {
    app().insertAdjacentHTML("beforeend", `<form class="card stack" id="del" style="margin-top:14px;max-width:520px"><h2>${t("Delete my account")}</h2>
      <p class="muted small">${t("Removes your name, email, phone, photos, profile and open requests. Orders and payment records are kept without your personal details.")}</p>
      <div><label for="dp">${t("Password")}</label><input id="dp" name="password" type="password" required autocomplete="current-password"></div>
      <div><button class="secondary">${t("Delete my account")}</button></div></form>`);
    bind("#del", "submit", (e, f) => act(async () => {
      if (!confirm(t("Delete your account permanently? This can't be undone."))) return;
      await api("DELETE", "/api/me", { password: f.password.value });
      setSession("", null); go("#/"); flash(t("Your account was deleted"));
    }));
  }
  if (tp) {
    const picker = pickerMap($("#map"), loc, p => { loc = p; });
    $("#locate").addEventListener("click", () => picker.locate());
    bind("#tp", "submit", (e, f) => act(async () => {
      if (!loc) throw new Error(t("Set your base location on the map"));
      await api("PUT", "/api/me/technician-profile", {
        bio: f.bio.value, city: f.city.value, serviceRadiusKm: +f.radius.value, location: loc,
        categoryIds: $$("input[name=cat]:checked", f).map(i => +i.value),
      });
    }, t("Profile saved")));
  }
}

async function leaderboard(params) {
  const role = params.get("role") === "seller" ? "seller" : "technician";
  const b = await api("GET", "/api/leaderboard?role=" + role);
  app().innerHTML = `<h1>${t("Leaderboard")}</h1>
    <div class="tabs" role="group"><button class="${role === "technician" ? "on" : ""}" aria-pressed="${role === "technician"}" data-go="#/leaderboard?role=technician">${t("Technicians")}</button>
      <button class="${role === "seller" ? "on" : ""}" aria-pressed="${role === "seller"}" data-go="#/leaderboard?role=seller">${t("Sellers")}</button></div>
    <p class="muted">${free() ? t("Promoted placement opens when online payments launch. During the free beta nobody can pay to be first.") : t("Ranked by boost paid in the last {days} days. Boosted profiles appear first in search and are labelled “Promoted”. Want to be #1? Pay more than the current top on your {link} page.", { days: b.boostDays, link: `<a href="#/account">${t("Account")}</a>` })}</p>
    ${b.entries.length ? `<table><tr><th>#</th><th>${t("Name")}</th><th></th><th>${t("Boost")}</th></tr>
      ${b.entries.map(e => `<tr><td>${e.rank}</td><td><strong>${esc(e.name)}</strong></td><td class="muted">${esc(detailText(e.detail))}</td><td>${gel(e.boostTetri)}</td></tr>`).join("")}</table>`
      : emptyState(t("Nobody is promoted yet."), `<a class="btn secondary" href="#/register?role=${role}">${joinLabel(role)}</a>`)}`;
}

function login() {
  app().innerHTML = `<form class="card stack" id="f" style="max-width:420px;margin:auto"><h1>${t("Log in")}</h1>
    <div><label for="le">${t("Email")}</label><input id="le" name="email" type="email" required autocomplete="email"></div>
    <div><label for="lpw">${t("Password")}</label><input id="lpw" name="password" type="password" required autocomplete="current-password"></div>
    <button>${t("Log in")}</button><p class="muted">${t("No account?")} <a href="#/register">${t("Sign up")}</a></p></form>`;
  bind("#f", "submit", (e, f) => act(async () => {
    const r = await api("POST", "/api/auth/login", { email: f.email.value, password: f.password.value });
    setSession(r.token, r.user); go("#/");
  }));
}

function register(params) {
  const pre = params.get("role");
  const fee = free() ? t("free during the beta") : t("paid registration");
  app().innerHTML = `<form class="card stack" id="f" style="max-width:520px;margin:auto"><h1>${t("Create account")}</h1>
    <div><label for="rr">${t("I am a…")}</label><select id="rr" name="role">
      <option value="customer">${t("Customer: I need something fixed or want to buy parts")}</option>
      <option value="technician" ${pre === "technician" ? "selected" : ""}>${t("Technician: I fix things ({fee})", { fee })}</option>
      <option value="seller" ${pre === "seller" ? "selected" : ""}>${t("Seller: I sell parts/appliances ({fee})", { fee })}</option></select></div>
    <div><label for="rn">${t("Full name")}</label><input id="rn" name="name" required minlength="2" autocomplete="name"></div>
    <div><label for="re">${t("Email")}</label><input id="re" name="email" type="email" required autocomplete="email"></div>
    <div><label for="rp">${t("Phone")}</label><input id="rp" name="phone" type="tel" inputmode="tel" autocomplete="tel" required placeholder="+995 5XX XX XX XX" aria-describedby="rph">
      <div class="hint" id="rph">${t("Georgian number, e.g. 555 12 34 56")}</div></div>
    <div><label for="rpw">${t("Password (8+ characters)")}</label><input id="rpw" name="password" type="password" required minlength="8" autocomplete="new-password"></div>
    <label class="check"><input type="checkbox" name="acceptTerms" required> <span>${t("I accept the {link}", { link: `<a href="${termsURL()}" target="_blank">${t("Terms & Conditions")}</a>` })}</span></label>
    <button>${t("Create account")}</button></form>`;
  bind("#f", "submit", (e, f) => act(async () => {
    if (!validPhone(f.phone.value)) throw new Error(t("Enter a Georgian phone number, e.g. +995 555 12 34 56"));
    const r = await api("POST", "/api/auth/register", {
      role: f.role.value, name: f.name.value, email: f.email.value, phone: f.phone.value, password: f.password.value, acceptTerms: f.acceptTerms.checked,
    });
    setSession(r.token, r.user);
    go(r.user.role === "seller" ? "#/seller" : r.user.role === "technician" ? "#/account" : "#/");
  }, t("Welcome to Eliava!")));
}

async function admin() {
  if (loginFirst(["admin"])) return;
  const [reveals, flagged, reports] = await Promise.all([api("GET", "/api/admin/reveals"), api("GET", "/api/admin/flagged-posts"), api("GET", "/api/admin/reports")]);
  app().innerHTML = `<h1>${t("Admin")}</h1>
    <p><a class="btn secondary" href="#/eliava">${t("Edit Eliava stall pins")}</a></p>
    <h2>${t("Reports from users")} (${reports.length})</h2>
    <table><tr><th>${t("When")}</th><th>${t("What")}</th><th>${t("Reason")}</th><th>${t("Reported by")}</th><th></th></tr>
      ${reports.map(r => `<tr><td>${when(r.createdAt)}</td><td>${t("target:" + r.targetType)} #${r.targetId}<br><strong>${esc(r.targetLabel)}</strong>${r.reportCount > 1 ? ` <span class="badge warn">${t("{n} reports", { n: r.reportCount })}</span>` : ""}</td>
        <td>${t("reason:" + r.reason)}${r.details ? `<br><span class="muted small">${esc(r.details)}</span>` : ""}</td><td>${esc(r.reporterName)}</td>
        <td><button class="secondary small" data-resolve="${r.id}">${t("Mark resolved")}</button></td></tr>`).join("") || `<tr><td colspan="5" class="muted">${t("No open reports.")}</td></tr>`}</table>
    <h2>${t("Who viewed which customer number")}</h2>
    <table><tr><th>${t("When")}</th><th>${t("Technician")}</th><th>${t("Request")}</th><th>${t("Customer")}</th><th>${t("Paid")}</th></tr>
      ${reveals.map(r => `<tr><td>${when(r.createdAt)}</td><td>${esc(r.technicianName)}<br><span class="muted small">${esc(r.technicianPhone)} · ${esc(r.technicianEmail)}</span></td>
        <td>#${r.postId} ${esc(r.postTitle)}</td><td>${esc(r.customerName)}</td><td>${gel(r.priceTetri)}</td></tr>`).join("") || `<tr><td colspan="5" class="muted">${t("No views yet.")}</td></tr>`}</table>
    <h2>${t("Flagged posts (possible duplicates)")}</h2>
    <div class="grid">${flagged.map(p => postCard(p, "admin")).join("") || `<p class="muted">${t("Nothing flagged.")}</p>`}</div>`;
  bind("[data-resolve]", "click", (e, el) => act(async () => { await api("POST", `/api/admin/reports/${el.dataset.resolve}/resolve`); admin(); }, t("Report resolved")));
}

/* ---------- router ---------- */
const routes = { "": home, technicians, post: newPost, "my-posts": myPosts, jobs, market, eliava, seller, orders, account, leaderboard, login, register, admin };
const titles = { "": "", technicians: "Find a technician", post: "Post a problem", "my-posts": "My requests", jobs: "Jobs near me", market: "Parts market", eliava: "Eliava market map", seller: "My shop", orders: "My orders", account: "My account", leaderboard: "Leaderboard", login: "Log in", register: "Create account", admin: "Admin" };
// Renders are queued so a slow page can't overwrite the one the user navigated to next.
let renderQueue = Promise.resolve();
let navigated = false;
function render() { renderQueue = renderQueue.then(renderNow); return renderQueue; }
async function renderNow() {
  const [name, query] = location.hash.replace(/^#\/?/, "").split("?");
  renderNav();
  const page = routes[name] ? name : "";
  document.title = titles[page] ? `${t(titles[page])} · Eliava` : t("Eliava: technicians, repairs and parts in Georgia");
  document.body.classList.add("busy"); app().setAttribute("aria-busy", "true");
  try { await routes[page](new URLSearchParams(query || "")); window.scrollTo(0, 0); }
  catch (e) { app().innerHTML = `<div class="card"><h1>${t("Something went wrong")}</h1><p class="muted">${esc(e.message)}</p></div>`; }
  finally { document.body.classList.remove("busy"); app().removeAttribute("aria-busy"); }
  if (navigated) { const h = $("h1", app()); if (h) { h.tabIndex = -1; h.focus({ preventScroll: true }); } }
}
window.addEventListener("hashchange", () => { navigated = true; render(); });
document.addEventListener("click", e => {
  const g = e.target.closest("[data-go]"); if (g) go(g.dataset.go);
  const r = e.target.closest("[data-report]"); if (r) { const [type, id] = r.dataset.report.split(":"); openReport(type, +id); }
});
(async () => {
  applyStaticText();
  try {
    [S.config, S.cats] = await Promise.all([api("GET", "/api/config"), api("GET", "/api/categories")]);
  } catch (e) {
    app().innerHTML = `<div class="card"><h1>${t("Can't reach the server")}</h1><p class="muted">${esc(e.message)}</p></div>`;
    return;
  }
  await refreshMe();
  render();
})();
