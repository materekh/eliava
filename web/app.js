"use strict";
const API = (window.ELIAVA_API || "").replace(/\/$/, "");
const TBILISI = [41.7151, 44.8271];
const ELIAVA = [41.7271, 44.7784];
const S = { token: localStorage.getItem("eliava_token") || "", me: null, config: {}, cats: [] };
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
const app = () => $("#app");

function esc(s) {
  return String(s ?? "").replace(/[&<>"']/g, c => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}
function gel(t) { const v = (t || 0) / 100; return (Number.isInteger(v) ? v : v.toFixed(2)) + " ₾"; }
function when(d) { return new Date(d).toLocaleString(); }
function flash(msg, isErr) {
  const el = document.createElement("div");
  el.className = "flash" + (isErr ? " error" : "");
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), isErr ? 6000 : 3500);
}
async function api(method, path, body) {
  const opt = { method, headers: {} };
  if (S.token) opt.headers.Authorization = "Bearer " + S.token;
  if (body instanceof FormData) opt.body = body;
  else if (body !== undefined) { opt.headers["Content-Type"] = "application/json"; opt.body = JSON.stringify(body); }
  const res = await fetch(API + path, opt);
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    if (res.status === 401 && S.token) setSession("", null);
    throw new Error(data.error || "Request failed (" + res.status + ")");
  }
  return data;
}
const mediaURL = url => API + url;
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
function bind(sel, ev, fn) { $$(sel).forEach(el => el.addEventListener(ev, e => { if (ev === "submit") e.preventDefault(); fn(e, el); })); }
function cats(kind) { return S.cats.filter(c => c.kind === kind); }
function catOptions(kind, selected, allLabel) {
  return (allLabel ? `<option value="">${esc(allLabel)}</option>` : "") +
    cats(kind).map(c => `<option value="${allLabel ? c.slug : c.id}" ${String(selected) === String(allLabel ? c.slug : c.id) ? "selected" : ""}>${esc(c.name)}</option>`).join("");
}
function thumbs(media) {
  if (!media || !media.length) return "";
  return `<div class="thumbs">${media.map(m => m.kind === "video"
    ? `<video src="${esc(mediaURL(m.url))}" controls preload="metadata"></video>`
    : `<a href="${esc(mediaURL(m.url))}" target="_blank"><img src="${esc(mediaURL(m.url))}" alt=""></a>`).join("")}</div>`;
}
function baseMap(el, center, zoom) {
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
      if (!navigator.geolocation) return flash("Your browser can't share location", true);
      navigator.geolocation.getCurrentPosition(p => set(p.coords.latitude, p.coords.longitude, true), () => flash("Could not get your location — click the map instead", true));
    },
  };
}
function loginFirst(roles) {
  if (!S.me) {
    app().innerHTML = `<div class="card"><h2>Please log in</h2><p class="muted">You need an account for this page.</p>
      <a class="btn" href="#/login">Log in</a> <a class="btn secondary" href="#/register">Create account</a></div>`;
    return true;
  }
  if (roles && S.me.role !== "admin" && !roles.includes(S.me.role)) {
    app().innerHTML = `<div class="card"><h2>Not available</h2><p class="muted">This page is for ${esc(roles.join(" / "))} accounts. You are logged in as a ${esc(S.me.role)}.</p></div>`;
    return true;
  }
  return false;
}
function registrationNotice() {
  if (!S.me || S.me.registrationActive || !["technician", "seller"].includes(S.me.role)) return "";
  return `<p class="notice">Your ${esc(S.me.role)} registration isn't paid yet, so you are hidden from customers. Pay ${gel(S.config.registrationFee)} for ${S.config.registrationDays} days on the <a href="#/account">Account</a> page.</p>`;
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
  $("#nav").innerHTML = links.map(([h, t]) => `<a href="${h}" class="${h === cur ? "active" : ""}">${t}</a>`).join("");
  $("#userbox").innerHTML = S.me
    ? `<a href="#/account" style="color:#fff">${esc(S.me.name)}</a> <span class="badge">${esc(S.me.role)}</span> <span>${gel(S.me.balanceTetri)}</span> <button class="secondary" id="logout">Log out</button>`
    : `<a class="btn secondary" href="#/login">Log in</a> <a class="btn" href="#/register">Sign up</a>`;
  $("#logout")?.addEventListener("click", async () => { await act(() => api("POST", "/api/auth/logout")); setSession("", null); go("#/"); });
}
async function refreshMe() { if (S.token) { S.me = await api("GET", "/api/me").catch(() => null); renderNav(); } }

/* ---------- pages ---------- */
async function home() {
  const [tech, sell] = await Promise.all([api("GET", "/api/leaderboard?role=technician"), api("GET", "/api/leaderboard?role=seller")]);
  const top = b => b.entries.length ? b.entries.slice(0, 5).map(e => `<li><strong>${esc(e.name)}</strong> <span class="muted">${esc(e.detail)}</span></li>`).join("") : `<li class="muted">Nobody yet — be the first!</li>`;
  app().innerHTML = `
    <div class="hero"><h1>Repairs, technicians and parts — all in one place</h1>
      <p>Find a nearby technician, post what's broken, or buy parts from the Eliava market with delivery.</p></div>
    <div class="grid-3">
      <div class="card service"><h3>🔧 Find a technician</h3><p class="muted">Search by speciality and distance from you.</p><a class="btn" href="#/technicians">Search technicians</a></div>
      <div class="card service"><h3>📝 Post a problem</h3><p class="muted">Describe it, add photos/videos and your location. Get a price estimate; nearby technicians contact you.</p><a class="btn" href="#/post">Post a problem</a></div>
      <div class="card service"><h3>🛒 Parts market</h3><p class="muted">Parts and appliances from Eliava market sellers, with stall numbers on the map and home delivery.</p><a class="btn" href="#/market">Open market</a></div>
    </div>
    <div class="grid-3" style="margin-top:14px">
      <div class="card"><h3>Top technicians</h3><ol>${top(tech)}</ol></div>
      <div class="card"><h3>Top sellers</h3><ol>${top(sell)}</ol></div>
    </div>`;
}

async function technicians(params) {
  let loc = null;
  app().innerHTML = `<h1>Find a technician</h1>
    <div class="card stack">
      <div class="row">
        <div><label>Speciality</label><select id="cat">${catOptions("repair", params.get("category") || "", "All specialities")}</select></div>
        <div><label>Within</label><select id="radius"><option>5</option><option selected>10</option><option>25</option><option>50</option><option>200</option></select></div>
        <div><button class="secondary" id="locate">📍 Use my location</button></div>
        <div><button id="search">Search</button></div>
      </div>
      <p class="muted small">Click the map to set where you need the repair.</p>
      <div id="map" class="map"></div>
    </div>
    <div id="results" class="grid" style="margin-top:14px"></div>`;
  let markers = [];
  const picker = pickerMap($("#map"), null, p => { loc = p; });
  $("#locate").onclick = () => picker.locate();
  const search = async () => {
    const q = new URLSearchParams({ category: $("#cat").value, radius_km: $("#radius").value });
    if (loc) { q.set("lat", loc.lat); q.set("lng", loc.lng); }
    const list = await act(() => api("GET", "/api/technicians?" + q));
    if (!list) return;
    markers.forEach(m => m.remove());
    markers = list.filter(t => t.location).map(t => L.circleMarker([t.location.lat, t.location.lng], { radius: 8, color: "#0e7c66" }).addTo(picker.map).bindPopup(esc(t.name)));
    $("#results").innerHTML = list.length ? list.map(t => `
      <div class="card">
        <h3>${esc(t.name)} ${t.boostTetri > 0 ? `<span class="badge top">★ Top</span>` : ""}</h3>
        <div>${t.categories.map(c => `<span class="badge">${esc(c.name)}</span>`).join("")}</div>
        <p class="muted">${esc(t.city)}${t.distanceKm != null ? ` · ${t.distanceKm.toFixed(1)} km away` : ""} · works within ${t.serviceRadiusKm} km</p>
        <p>${esc(t.bio)}</p>
        <a class="btn" href="tel:${esc(t.phone)}">📞 ${esc(t.phone)}</a>
      </div>`).join("") : `<p class="muted">No technicians found. Try a bigger distance or another speciality.</p>`;
  };
  $("#search").onclick = search;
  search();
}

async function newPost() {
  if (loginFirst(["customer"])) return;
  let loc = null;
  app().innerHTML = `<h1>Post a problem</h1>
    <form class="card stack" id="f">
      <div class="row">
        <div><label>What needs fixing?</label><select name="categoryId" required>${catOptions("repair")}</select></div>
        <div><label>Short title</label><input name="title" required minlength="5" maxlength="120" placeholder="e.g. AC blows warm air"></div>
      </div>
      <div><label>Description (at least 20 characters)</label><textarea name="description" required minlength="20" placeholder="Brand, model, what happens, since when…"></textarea></div>
      <label class="check"><input type="checkbox" name="hasParts" value="true"> I already have the needed parts at home</label>
      <div><label>Photos and videos (at least one photo; photos up to 10 MB, videos up to 100 MB)</label><input type="file" name="media" multiple accept="image/*,video/*" required></div>
      <div><label>Address (optional — technicians only see the area until they view your number)</label><input name="address" maxlength="300"></div>
      <div class="row"><div><label>Location — click the map</label></div><div style="flex:0"><button type="button" class="secondary" id="locate">📍 Use my location</button></div></div>
      <div id="map" class="map"></div>
      <div id="estimate"></div>
      <div class="row"><div style="flex:0"><button type="button" class="secondary" id="est">Get price estimate</button></div><div style="flex:0"><button type="submit">Post request</button></div></div>
    </form>`;
  const picker = pickerMap($("#map"), null, p => { loc = p; });
  $("#locate").onclick = () => picker.locate();
  const f = $("#f");
  const showEstimate = e => {
    $("#estimate").innerHTML = `<div class="estimate"><strong>Estimated cost: ${gel(e.minTetri)} – ${gel(e.maxTetri)}</strong><br><span class="muted">${esc(e.note)}${e.source === "ai" ? " (AI estimate)" : ""}</span></div>`;
  };
  $("#est").onclick = () => act(async () => {
    if (!loc) throw new Error("Pick your location on the map first");
    showEstimate(await api("POST", "/api/estimate", { categoryId: +f.categoryId.value, description: f.description.value, hasParts: f.hasParts.checked, location: loc }));
  });
  bind("#f", "submit", () => act(async () => {
    if (!loc) throw new Error("Pick your location on the map");
    const fd = new FormData(f);
    fd.set("lat", loc.lat); fd.set("lng", loc.lng); fd.set("hasParts", f.hasParts.checked ? "true" : "false");
    const btn = $("button[type=submit]", f); btn.disabled = true; btn.textContent = "Uploading…";
    try {
      const p = await api("POST", "/api/posts", fd);
      if (p.flagged) flash("Posted, but it looks like a repeat of one of your earlier requests, so it is hidden from technicians until an admin reviews it.", true);
      else flash("Your request is live! Estimated " + gel(p.estimate.minTetri) + " – " + gel(p.estimate.maxTetri));
      go("#/my-posts");
    } finally { btn.disabled = false; btn.textContent = "Post request"; }
  }));
}

function postCard(p, mode) {
  const status = p.status === "open" ? `<span class="badge ok">open</span>` : `<span class="badge">closed</span>`;
  const flagged = p.flagged ? `<span class="badge warn">flagged: ${esc(p.flagReason)}</span>` : "";
  let actions = "";
  if (mode === "owner") {
    actions = `<p class="muted">${p.viewCount} technician(s) viewed your number</p>
      <button class="secondary" data-viewers="${p.id}">Who viewed?</button>
      ${p.status === "open" ? `<button class="secondary" data-close="${p.id}">Mark solved / close</button>` : ""}
      <div id="viewers-${p.id}"></div>`;
  } else if (mode === "tech") {
    actions = p.contact
      ? `<p><strong>${esc(p.contact.name)}</strong> · <a href="tel:${esc(p.contact.phone)}">📞 ${esc(p.contact.phone)}</a>${p.address ? `<br><span class="muted">${esc(p.address)}</span>` : ""}</p>`
      : `<button class="accent" data-reveal="${p.id}">View number (${gel(S.config.revealPrice)})</button>`;
  }
  return `<div class="card">
    <h3>${esc(p.title)}</h3>
    <div>${status}<span class="badge">${esc(p.category.name)}</span>${p.hasParts ? `<span class="badge">has parts</span>` : ""}${flagged}</div>
    <p>${esc(p.description)}</p>
    ${thumbs(p.media)}
    <p class="muted">Estimate ${gel(p.estimate.minTetri)} – ${gel(p.estimate.maxTetri)}${p.distanceKm != null ? ` · ${p.distanceKm.toFixed(1)} km away` : ""} · ${when(p.createdAt)}</p>
    ${actions}
  </div>`;
}

async function myPosts() {
  if (loginFirst(["customer"])) return;
  const posts = await api("GET", "/api/posts/mine");
  app().innerHTML = `<h1>My repair requests</h1><p><a class="btn" href="#/post">+ New request</a></p>
    <div class="grid">${posts.map(p => postCard(p, "owner")).join("") || `<p class="muted">No requests yet.</p>`}</div>`;
  bind("[data-viewers]", "click", (e, el) => act(async () => {
    const v = await api("GET", `/api/posts/${el.dataset.viewers}/viewers`);
    $("#viewers-" + el.dataset.viewers).innerHTML = v.length ? `<ul>${v.map(x => `<li>${esc(x.name)} — ${when(x.viewedAt)}</li>`).join("")}</ul>` : `<p class="muted">Nobody yet.</p>`;
  }));
  bind("[data-close]", "click", (e, el) => act(async () => { await api("POST", `/api/posts/${el.dataset.close}/close`); myPosts(); }, "Request closed"));
}

async function jobs() {
  if (loginFirst(["technician"])) return;
  app().innerHTML = `<h1>Jobs near me</h1>${registrationNotice()}
    <div class="card row">
      <div><label>Category</label><select id="cat">${catOptions("repair", "", "All categories")}</select></div>
      <div><label>Within (km, empty = your service radius)</label><input id="radius" type="number" min="1" max="500"></div>
      <div><label class="check"><input type="checkbox" id="mine" checked> Only my specialities</label></div>
      <div><button id="load">Refresh</button></div>
    </div>
    <div id="list" class="grid" style="margin-top:14px"></div>`;
  const load = async () => {
    const q = new URLSearchParams({ category: $("#cat").value, mine: $("#mine").checked ? "1" : "0" });
    if ($("#radius").value) q.set("radius_km", $("#radius").value);
    const posts = await act(() => api("GET", "/api/posts/nearby?" + q));
    if (!posts) { $("#list").innerHTML = `<p class="muted">Set your location and specialities on the <a href="#/account">Account</a> page.</p>`; return; }
    $("#list").innerHTML = posts.map(p => postCard(p, "tech")).join("") || `<p class="muted">No open requests nearby right now.</p>`;
    bind("[data-reveal]", "click", (e, el) => act(async () => {
      if (!confirm(`View this customer's number for ${gel(S.config.revealPrice)}? You are charged once per request.`)) return;
      await api("POST", `/api/posts/${el.dataset.reveal}/reveal`);
      await refreshMe(); load();
    }));
  };
  $("#load").onclick = load;
  load();
}

function listingCard(l) {
  const img = l.media.find(m => m.kind === "image");
  const stall = l.seller.stallCode ? `<a href="#/eliava?stall=${l.seller.stallId}" class="badge">Eliava stall ${esc(l.seller.stallCode)}</a>` : "";
  return `<div class="card">
    ${img ? `<img class="cover" src="${esc(mediaURL(img.url))}" alt="">` : `<div class="cover"></div>`}
    <h3>${esc(l.title)} ${l.boostTetri > 0 ? `<span class="badge top">★ Top</span>` : ""}</h3>
    <div><span class="badge">${esc(l.category.name)}</span>${stall}</div>
    <p class="muted">${esc(l.description)}</p>
    <p><span class="price">${gel(l.priceTetri)}</span> <span class="muted">· ${l.stock} in stock</span></p>
    <p class="muted small">${esc(l.seller.shopName)} · <a href="tel:${esc(l.seller.phone)}">${esc(l.seller.phone)}</a></p>
    <button data-order="${l.id}">Order</button>
    <form class="stack" id="order-${l.id}" hidden style="margin-top:10px">
      <div class="row"><div><label>Quantity</label><input name="quantity" type="number" min="1" max="${l.stock}" value="1"></div></div>
      <label class="check"><input type="checkbox" name="delivery"> Deliver to my home (+${gel(S.config.deliveryFee)})</label>
      <input name="deliveryAddress" placeholder="Delivery address">
      <button type="submit">Place order (pay on delivery / pickup)</button>
    </form>
  </div>`;
}
function bindOrders(reload) {
  bind("[data-order]", "click", (e, el) => {
    if (loginFirst(["customer", "technician"])) return;
    $("#order-" + el.dataset.order).hidden = false; el.hidden = true;
  });
  $$("form[id^=order-]").forEach(f => f.addEventListener("submit", e => {
    e.preventDefault();
    act(async () => {
      await api("POST", "/api/orders", { listingId: +f.id.slice(6), quantity: +f.quantity.value, delivery: f.delivery.checked, deliveryAddress: f.deliveryAddress.value });
      reload();
    }, "Order placed! The seller will contact you.");
  }));
}

async function market(params) {
  const stalls = await api("GET", "/api/stalls");
  app().innerHTML = `<h1>Parts market</h1>
    <div class="card row">
      <div><label>Search</label><input id="q" placeholder="e.g. compressor" value="${esc(params.get("q") || "")}"></div>
      <div><label>Category</label><select id="cat">${catOptions("part", params.get("category") || "", "All categories")}</select></div>
      <div><label>Eliava stall</label><select id="stall"><option value="">Any stall</option>${stalls.map(s => `<option value="${s.id}" ${params.get("stall") == s.id ? "selected" : ""}>${esc(s.code)}</option>`).join("")}</select></div>
      <div><button id="search">Search</button></div>
    </div>
    <div id="list" class="grid" style="margin-top:14px"></div>`;
  const load = async () => {
    const q = new URLSearchParams({ q: $("#q").value, category: $("#cat").value, stall: $("#stall").value });
    const list = await act(() => api("GET", "/api/listings?" + q));
    if (!list) return;
    $("#list").innerHTML = list.map(listingCard).join("") || `<p class="muted">Nothing found.</p>`;
    bindOrders(load);
  };
  $("#search").onclick = load;
  $("#q").addEventListener("keydown", e => { if (e.key === "Enter") load(); });
  load();
}

async function eliava(params) {
  const stalls = await api("GET", "/api/stalls");
  const isAdmin = S.me?.role === "admin";
  app().innerHTML = `<h1>Eliava market map</h1>
    <p class="muted">Click a stall to see what its sellers offer.${isAdmin ? " <strong>Admin:</strong> drag pins to their real positions; changes save automatically." : ""}</p>
    <div id="map" class="map tall"></div>
    ${isAdmin ? `<form class="card row" id="add" style="margin-top:12px"><div><label>New stall code</label><input name="code" required placeholder="E1"></div><div><label>Name</label><input name="name"></div><div><button>Add stall at map centre</button></div></form>` : ""}
    <h2 id="title"></h2><div id="list" class="grid"></div>`;
  const map = baseMap($("#map"), ELIAVA, 17);
  const showStall = async st => {
    $("#title").textContent = "Stall " + st.code + (st.name ? " — " + st.name : "");
    const list = await act(() => api("GET", "/api/listings?stall=" + st.id));
    if (!list) return;
    $("#list").innerHTML = list.map(listingCard).join("") || `<p class="muted">No listings at this stall yet.</p>`;
    bindOrders(() => showStall(st));
  };
  stalls.forEach(st => {
    const m = L.marker([st.location.lat, st.location.lng], { draggable: isAdmin }).addTo(map)
      .bindTooltip(st.code, { permanent: true, direction: "top", offset: [-15, -12] });
    m.on("click", () => showStall(st));
    if (isAdmin) m.on("dragend", () => act(() => api("PATCH", "/api/admin/stalls/" + st.id, { location: m.getLatLng() }), "Stall " + st.code + " moved"));
  });
  const pre = stalls.find(s => String(s.id) === params.get("stall"));
  if (pre) { map.setView([pre.location.lat, pre.location.lng], 18); showStall(pre); }
  bind("#add", "submit", (e, f) => act(async () => {
    await api("POST", "/api/admin/stalls", { code: f.code.value, name: f.name.value, location: map.getCenter() });
    eliava(params);
  }, "Stall added"));
}

async function seller() {
  if (loginFirst(["seller"])) return;
  const [profile, stalls, listings, orders] = await Promise.all([
    S.me.role === "seller" ? api("GET", "/api/me/seller-profile") : {}, api("GET", "/api/stalls"),
    api("GET", "/api/listings/mine"), api("GET", "/api/orders/incoming")]);
  let pin = profile.stallId ? null : profile.location;
  app().innerHTML = `<h1>My shop</h1>${registrationNotice()}
    <form class="card stack" id="profile">
      <h3>Shop details</h3>
      <div class="row">
        <div><label>Shop name</label><input name="shopName" required value="${esc(profile.shopName)}"></div>
        <div><label>Eliava stall</label><select name="stallId"><option value="">Not at Eliava (pin on map below)</option>${stalls.map(s => `<option value="${s.id}" ${profile.stallId === s.id ? "selected" : ""}>${esc(s.code)}</option>`).join("")}</select></div>
        <div><label>Address</label><input name="address" value="${esc(profile.address)}"></div>
      </div>
      <p class="muted small">Not at Eliava? Click the map to pin your shop.</p>
      <div id="map" class="map" style="height:240px"></div>
      <div><button>Save shop</button></div>
    </form>
    <form class="card stack" id="new" style="margin-top:14px">
      <h3>New listing</h3>
      <div class="row">
        <div><label>Category</label><select name="categoryId">${catOptions("part")}</select></div>
        <div><label>Title</label><input name="title" required minlength="3"></div>
        <div><label>Price (₾)</label><input name="price" required inputmode="decimal" placeholder="25.50"></div>
        <div><label>In stock</label><input name="stock" type="number" min="0" value="1" required></div>
      </div>
      <div><label>Description</label><textarea name="description"></textarea></div>
      <div><label>Photos</label><input type="file" name="media" multiple accept="image/*,video/*"></div>
      <div><button>Publish listing</button></div>
    </form>
    <h2>My listings</h2>
    <table><tr><th>Item</th><th>Price (₾)</th><th>Stock</th><th>Visible</th><th></th></tr>
      ${listings.map(l => `<tr data-row="${l.id}"><td>${esc(l.title)}<br><span class="muted small">${esc(l.category.name)}</span></td>
        <td><input name="price" value="${l.priceTetri / 100}" style="width:90px"></td>
        <td><input name="stock" type="number" min="0" value="${l.stock}" style="width:80px"></td>
        <td><input name="active" type="checkbox" ${l.active ? "checked" : ""}></td>
        <td><button class="secondary" data-save="${l.id}">Save</button></td></tr>`).join("") || `<tr><td colspan="5" class="muted">No listings yet.</td></tr>`}
    </table>
    <h2>Incoming orders</h2>${ordersTable(orders, "seller")}`;
  const picker = pickerMap($("#map"), profile.location, p => { pin = p; });
  picker.map.setView(profile.location ? [profile.location.lat, profile.location.lng] : ELIAVA, 15);
  bind("#profile", "submit", (e, f) => act(async () => {
    const stallId = f.stallId.value ? +f.stallId.value : null;
    await api("PUT", "/api/me/seller-profile", { shopName: f.shopName.value, stallId, address: f.address.value, location: stallId ? null : pin });
  }, "Shop saved"));
  bind("#new", "submit", (e, f) => act(async () => { await api("POST", "/api/listings", new FormData(f)); seller(); }, "Listing published"));
  bind("[data-save]", "click", (e, el) => act(async () => {
    const row = $(`[data-row="${el.dataset.save}"]`);
    const price = Math.round(parseFloat($("[name=price]", row).value.replace(",", ".")) * 100);
    if (!(price > 0)) throw new Error("Enter a valid price");
    await api("PATCH", "/api/listings/" + el.dataset.save, { priceTetri: price, stock: +$("[name=stock]", row).value, active: $("[name=active]", row).checked });
  }, "Saved"));
  bindOrderStatus(seller);
}

const statusLabel = { pending: "Pending", confirmed: "Confirmed", out_for_delivery: "Out for delivery", completed: "Completed", cancelled: "Cancelled" };
const sellerNext = { pending: ["confirmed", "cancelled"], confirmed: ["out_for_delivery", "completed", "cancelled"], out_for_delivery: ["completed", "cancelled"] };
function ordersTable(orders, mode) {
  if (!orders.length) return `<p class="muted">No orders yet.</p>`;
  return `<table><tr><th>#</th><th>Item</th><th>${mode === "seller" ? "Customer" : "Seller"}</th><th>Qty</th><th>Total</th><th>Delivery</th><th>Status</th></tr>
    ${orders.map(o => {
      const btns = mode === "seller" ? (sellerNext[o.status] || []) : (o.status === "pending" ? ["cancelled"] : []);
      return `<tr><td>${o.id}</td><td>${esc(o.listingTitle)}</td>
        <td>${mode === "seller" ? `${esc(o.customerName)}<br><a href="tel:${esc(o.customerPhone)}">${esc(o.customerPhone)}</a>` : `${esc(o.sellerName)}${o.stallCode ? ` (stall ${esc(o.stallCode)})` : ""}<br><a href="tel:${esc(o.sellerPhone)}">${esc(o.sellerPhone)}</a>`}</td>
        <td>${o.quantity}</td><td>${gel(o.totalTetri)}</td><td>${o.delivery ? esc(o.deliveryAddress) : "Pickup"}</td>
        <td><span class="badge">${statusLabel[o.status]}</span><br>${btns.map(s => `<button class="secondary small" data-status="${s}" data-oid="${o.id}">${s === "cancelled" ? "Cancel" : statusLabel[s]}</button>`).join(" ")}</td></tr>`;
    }).join("")}</table>`;
}
function bindOrderStatus(reload) {
  bind("[data-status]", "click", (e, el) => act(async () => { await api("PATCH", "/api/orders/" + el.dataset.oid, { status: el.dataset.status }); reload(); }, "Order updated"));
}

async function orders() {
  if (loginFirst()) return;
  const list = await api("GET", "/api/orders/mine");
  app().innerHTML = `<h1>My orders</h1>${ordersTable(list, "buyer")}`;
  bindOrderStatus(orders);
}

async function account() {
  if (loginFirst()) return;
  const w = await api("GET", "/api/wallet");
  S.me = w.user; renderNav();
  const u = S.me, pro = ["technician", "seller"].includes(u.role);
  const board = pro ? await api("GET", "/api/leaderboard?role=" + u.role) : null;
  const tp = u.role === "technician" ? await api("GET", "/api/me/technician-profile") : null;
  const topAmount = board?.entries[0]?.boostTetri || 0;
  let loc = tp?.location || null;
  app().innerHTML = `<h1>My account</h1>
    <div class="grid-3">
      <div class="card"><h3>${esc(u.name)}</h3><p class="muted">${esc(u.email)} · ${esc(u.phone)}<br>Account type: ${esc(u.role)}</p></div>
      <div class="card"><h3>Wallet: ${gel(u.balanceTetri)}</h3>
        <form class="row" id="topup"><div><label>Top up (₾)</label><input name="amount" type="number" min="1" max="1000" value="20"></div><div><button>Top up</button></div></form>
        ${S.config.paymentsMode === "mock" ? `<p class="muted small">Test mode: no real money is charged.</p>` : ""}</div>
      ${pro ? `<div class="card"><h3>Registration</h3>
        <p>${u.registrationActive ? `<span class="badge ok">Active</span> until ${when(u.registrationPaidUntil)}` : `<span class="badge warn">Not paid</span> — you're hidden from customers`}</p>
        <button id="reg">Pay ${gel(S.config.registrationFee)} for ${S.config.registrationDays} days</button></div>` : ""}
    </div>
    ${pro ? `<div class="card" style="margin-top:14px"><h3>Leaderboard boost</h3>
      <p class="muted">Your active boost: ${gel(w.activeBoostTetri)}. Current #1 has ${gel(topAmount)} — pay more than that to get to the top for ${S.config.boostDays} days.</p>
      <form class="row" id="boost"><div><label>Boost amount (₾)</label><input name="amount" type="number" min="1" value="${Math.max(5, Math.floor((topAmount - w.activeBoostTetri) / 100) + 1)}"></div><div><button class="accent">Boost</button></div></form></div>` : ""}
    ${tp ? `<form class="card stack" id="tp" style="margin-top:14px"><h3>Technician profile</h3>
      <div class="row"><div><label>City</label><input name="city" value="${esc(tp.city)}"></div>
        <div><label>I travel up to (km)</label><input name="radius" type="number" min="1" max="500" value="${tp.serviceRadiusKm}"></div></div>
      <div><label>About me</label><textarea name="bio">${esc(tp.bio)}</textarea></div>
      <div><label>Specialities</label>${cats("repair").map(c => `<label class="check"><input type="checkbox" name="cat" value="${c.id}" ${tp.categories.some(x => x.id === c.id) ? "checked" : ""}> ${esc(c.name)}</label>`).join("")}</div>
      <div class="row"><div><label>My base location — click the map</label></div><div style="flex:0"><button type="button" class="secondary" id="locate">📍 Use my location</button></div></div>
      <div id="map" class="map"></div>
      <div><button>Save profile</button></div></form>` : ""}
    <h2>Wallet history</h2>
    <table><tr><th>When</th><th>What</th><th>Amount</th></tr>${w.transactions.map(t => `<tr><td>${when(t.createdAt)}</td><td>${esc(t.reason)}</td><td>${t.amountTetri > 0 ? "+" : ""}${gel(t.amountTetri)}</td></tr>`).join("") || `<tr><td colspan="3" class="muted">Nothing yet.</td></tr>`}</table>`;
  bind("#topup", "submit", (e, f) => act(async () => { await api("POST", "/api/wallet/topup", { amountTetri: Math.round(+f.amount.value * 100) }); account(); }, "Wallet topped up"));
  $("#reg")?.addEventListener("click", () => act(async () => { await api("POST", "/api/registration/pay"); account(); }, "Registration paid"));
  bind("#boost", "submit", (e, f) => act(async () => { await api("POST", "/api/boosts", { amountTetri: Math.round(+f.amount.value * 100) }); account(); }, "Boost active!"));
  if (tp) {
    const picker = pickerMap($("#map"), loc, p => { loc = p; });
    $("#locate").onclick = () => picker.locate();
    bind("#tp", "submit", (e, f) => act(async () => {
      if (!loc) throw new Error("Set your base location on the map");
      await api("PUT", "/api/me/technician-profile", {
        bio: f.bio.value, city: f.city.value, serviceRadiusKm: +f.radius.value, location: loc,
        categoryIds: $$("input[name=cat]:checked", f).map(i => +i.value),
      });
    }, "Profile saved"));
  }
}

async function leaderboard(params) {
  const role = params.get("role") === "seller" ? "seller" : "technician";
  const b = await api("GET", "/api/leaderboard?role=" + role);
  app().innerHTML = `<h1>Leaderboard</h1>
    <div class="tabs"><button class="${role === "technician" ? "on" : ""}" onclick="go('#/leaderboard?role=technician')">Technicians</button>
      <button class="${role === "seller" ? "on" : ""}" onclick="go('#/leaderboard?role=seller')">Sellers</button></div>
    <p class="muted">Ranked by boost paid in the last ${b.boostDays} days. Boosted ${role}s appear first in search. Want to be #1? Pay more than the current top on your <a href="#/account">Account</a> page.</p>
    <table><tr><th>#</th><th>Name</th><th></th><th>Boost</th></tr>
      ${b.entries.map(e => `<tr><td>${e.rank}</td><td><strong>${esc(e.name)}</strong></td><td class="muted">${esc(e.detail)}</td><td>${gel(e.boostTetri)}</td></tr>`).join("") || `<tr><td colspan="4" class="muted">Nobody has boosted yet.</td></tr>`}</table>`;
}

function login() {
  app().innerHTML = `<form class="card stack" id="f" style="max-width:420px;margin:auto"><h1>Log in</h1>
    <div><label>Email</label><input name="email" type="email" required></div>
    <div><label>Password</label><input name="password" type="password" required></div>
    <button>Log in</button><p class="muted">No account? <a href="#/register">Sign up</a></p></form>`;
  bind("#f", "submit", (e, f) => act(async () => {
    const r = await api("POST", "/api/auth/login", { email: f.email.value, password: f.password.value });
    setSession(r.token, r.user); go("#/");
  }));
}

function register() {
  app().innerHTML = `<form class="card stack" id="f" style="max-width:520px;margin:auto"><h1>Create account</h1>
    <div><label>I am a…</label><select name="role">
      <option value="customer">Customer — I need something fixed or want to buy parts</option>
      <option value="technician">Technician — I fix things (paid registration)</option>
      <option value="seller">Seller — I sell parts/appliances (paid registration)</option></select></div>
    <div><label>Full name</label><input name="name" required minlength="2"></div>
    <div><label>Email</label><input name="email" type="email" required></div>
    <div><label>Phone</label><input name="phone" required placeholder="+995 5xx xx xx xx"></div>
    <div><label>Password (8+ characters)</label><input name="password" type="password" required minlength="8"></div>
    <label class="check"><input type="checkbox" name="acceptTerms" required> I accept the <a href="terms.html" target="_blank">Terms &amp; Conditions</a></label>
    <button>Create account</button></form>`;
  bind("#f", "submit", (e, f) => act(async () => {
    const r = await api("POST", "/api/auth/register", {
      role: f.role.value, name: f.name.value, email: f.email.value, phone: f.phone.value, password: f.password.value, acceptTerms: f.acceptTerms.checked,
    });
    setSession(r.token, r.user);
    go(r.user.role === "technician" || r.user.role === "seller" ? (r.user.role === "seller" ? "#/seller" : "#/account") : "#/");
  }, "Welcome to Eliava!"));
}

async function admin() {
  if (loginFirst(["admin"])) return;
  const [reveals, flagged] = await Promise.all([api("GET", "/api/admin/reveals"), api("GET", "/api/admin/flagged-posts")]);
  app().innerHTML = `<h1>Admin</h1>
    <p><a class="btn secondary" href="#/eliava">Edit Eliava stall pins</a></p>
    <h2>Who viewed which customer number</h2>
    <table><tr><th>When</th><th>Technician</th><th>Request</th><th>Customer</th><th>Paid</th></tr>
      ${reveals.map(r => `<tr><td>${when(r.createdAt)}</td><td>${esc(r.technicianName)}<br><span class="muted small">${esc(r.technicianPhone)} · ${esc(r.technicianEmail)}</span></td>
        <td>#${r.postId} ${esc(r.postTitle)}</td><td>${esc(r.customerName)}</td><td>${gel(r.priceTetri)}</td></tr>`).join("") || `<tr><td colspan="5" class="muted">No views yet.</td></tr>`}</table>
    <h2>Flagged posts (possible duplicates)</h2>
    <div class="grid">${flagged.map(p => postCard(p, "admin")).join("") || `<p class="muted">Nothing flagged.</p>`}</div>`;
}

/* ---------- router ---------- */
const routes = { "": home, technicians, post: newPost, "my-posts": myPosts, jobs, market, eliava, seller, orders, account, leaderboard, login, register, admin };
// Renders are queued so a slow page can't overwrite the one the user navigated to next.
let renderQueue = Promise.resolve();
function render() { renderQueue = renderQueue.then(renderNow); return renderQueue; }
async function renderNow() {
  const [name, query] = location.hash.replace(/^#\/?/, "").split("?");
  renderNav();
  const page = routes[name] || home;
  try { await page(new URLSearchParams(query || "")); window.scrollTo(0, 0); }
  catch (e) { app().innerHTML = `<div class="card"><h2>Something went wrong</h2><p class="muted">${esc(e.message)}</p></div>`; }
}
window.addEventListener("hashchange", render);
(async () => {
  try {
    [S.config, S.cats] = await Promise.all([api("GET", "/api/config"), api("GET", "/api/categories")]);
  } catch (e) {
    app().innerHTML = `<div class="card"><h2>Can't reach the server</h2><p class="muted">${esc(e.message)}. Check that the Go API is running and that ELIAVA_API in config.js points to it.</p></div>`;
    return;
  }
  await refreshMe();
  render();
})();
