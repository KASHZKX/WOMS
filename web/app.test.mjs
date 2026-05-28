import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

class MiniClassList {
  constructor(element) {
    this.element = element;
  }

  add(...tokens) {
    const classes = new Set(this.element.className.split(/\s+/).filter(Boolean));
    tokens.forEach((token) => classes.add(token));
    this.element.className = Array.from(classes).join(" ");
  }

  remove(...tokens) {
    const classes = new Set(this.element.className.split(/\s+/).filter(Boolean));
    tokens.forEach((token) => classes.delete(token));
    this.element.className = Array.from(classes).join(" ");
  }

  contains(token) {
    return this.element.className.split(/\s+/).includes(token);
  }

  toggle(token, force) {
    const enabled = force ?? !this.contains(token);
    if (enabled) {
      this.add(token);
    } else {
      this.remove(token);
    }
    return enabled;
  }
}

class MiniElement {
  constructor(tagName, ownerDocument) {
    this.tagName = tagName.toUpperCase();
    this.ownerDocument = ownerDocument;
    this.children = [];
    this.parentNode = null;
    this.attributes = new Map();
    this.dataset = {};
    this.style = {};
    this.eventListeners = new Map();
    this.className = "";
    this.classList = new MiniClassList(this);
    this.hidden = false;
    this.value = "";
    this.textContent = "";
    this._innerHTML = "";
  }

  set id(value) {
    this.setAttribute("id", value);
  }

  get id() {
    return this.attributes.get("id") ?? "";
  }

  set innerHTML(value) {
    this._innerHTML = String(value);
    this.textContent = "";
    this.children = [];
    if (/<input\b/i.test(this._innerHTML)) {
      const input = this.ownerDocument.createElement("input");
      const valueMatch = this._innerHTML.match(/\bvalue="([^"]*)"/i);
      if (valueMatch) {
        input.value = valueMatch[1];
        input.setAttribute("value", valueMatch[1]);
      }
      this.appendChild(input);
    }
  }

  get innerHTML() {
    return this._innerHTML;
  }

  setAttribute(name, value) {
    const stringValue = String(value);
    this.attributes.set(name, stringValue);
    if (name === "id") {
      this.ownerDocument.elementsById.set(stringValue, this);
    } else if (name === "class") {
      this.className = stringValue;
    } else if (name === "name") {
      this.name = stringValue;
    } else if (name.startsWith("data-")) {
      this.dataset[dataKey(name)] = stringValue;
    } else if (name === "hidden") {
      this.hidden = true;
    }
  }

  getAttribute(name) {
    if (name === "class") {
      return this.className;
    }
    return this.attributes.get(name) ?? null;
  }

  removeAttribute(name) {
    this.attributes.delete(name);
    if (name === "hidden") {
      this.hidden = false;
    }
  }

  append(...nodes) {
    nodes.forEach((node) => this.appendChild(node));
  }

  appendChild(node) {
    node.parentNode = this;
    this.children.push(node);
    return node;
  }

  addEventListener(type, listener) {
    const listeners = this.eventListeners.get(type) ?? [];
    listeners.push(listener);
    this.eventListeners.set(type, listeners);
  }

  removeEventListener(type, listener) {
    const listeners = this.eventListeners.get(type) ?? [];
    this.eventListeners.set(type, listeners.filter((item) => item !== listener));
  }

  async dispatchEvent(event) {
    event.target ??= this;
    event.currentTarget = this;
    event.preventDefault ??= () => {
      event.defaultPrevented = true;
    };
    const listeners = this.eventListeners.get(event.type) ?? [];
    await Promise.all(listeners.map((listener) => listener(event)));
    return !event.defaultPrevented;
  }

  querySelector(selector) {
    return this.querySelectorAll(selector)[0] ?? null;
  }

  querySelectorAll(selector) {
    const selectors = selector.trim().split(/\s+/);
    if (selectors.length > 1) {
      const [first, ...rest] = selectors;
      return this.querySelectorAll(first).flatMap((node) => node.querySelectorAll(rest.join(" ")));
    }
    const matches = [];
    for (const child of walk(this)) {
      if (child !== this && matchesSelector(child, selector)) {
        matches.push(child);
      }
    }
    return matches;
  }

  closest(selector) {
    let node = this;
    while (node) {
      if (matchesSelector(node, selector)) {
        return node;
      }
      node = node.parentNode;
    }
    return null;
  }

  reset() {
    for (const child of walk(this)) {
      if ("value" in child) {
        child.value = child.getAttribute("value") ?? "";
      }
    }
  }

  showModal() {
    this.open = true;
    this.setAttribute("open", "");
  }

  close() {
    this.open = false;
    this.removeAttribute("open");
  }
}

class MiniDocument extends MiniElement {
  constructor() {
    super("#document", null);
    this.ownerDocument = this;
    this.elementsById = new Map();
    this.body = this.createElement("body");
    this.appendChild(this.body);
  }

  createElement(tagName) {
    return new MiniElement(tagName, this);
  }

  getElementById(id) {
    return this.elementsById.get(id) ?? null;
  }

  elementFromPoint() {
    return null;
  }
}

function* walk(root) {
  for (const child of root.children) {
    yield child;
    yield* walk(child);
  }
}

function dataKey(name) {
  return name.slice(5).replace(/-([a-z])/g, (_, char) => char.toUpperCase());
}

function matchesSelector(element, selector) {
  const trimmed = selector.trim();
  if (!trimmed) {
    return false;
  }
  if (trimmed.startsWith("#")) {
    return element.id === trimmed.slice(1);
  }
  if (trimmed.startsWith(".")) {
    return element.classList.contains(trimmed.slice(1));
  }
  if (trimmed.startsWith("[")) {
    const attr = trimmed.slice(1, -1).split("=")[0];
    return element.getAttribute(attr) !== null;
  }
  const attrMatch = trimmed.match(/^([a-z]+)\[([^=]+)="([^"]*)"\]$/i);
  if (attrMatch) {
    return element.tagName.toLowerCase() === attrMatch[1].toLowerCase()
      && element.getAttribute(attrMatch[2]) === attrMatch[3];
  }
  const compoundClassMatch = trimmed.match(/^\.([^.]+)\.([^.]+)$/);
  if (compoundClassMatch) {
    return element.classList.contains(compoundClassMatch[1])
      && element.classList.contains(compoundClassMatch[2]);
  }
  return element.tagName.toLowerCase() === trimmed.toLowerCase();
}

function appendElement(parent, tagName, attributes = {}) {
  const element = parent.ownerDocument.createElement(tagName);
  for (const [name, value] of Object.entries(attributes)) {
    if (name === "hidden") {
      element.hidden = Boolean(value);
      if (value) {
        element.setAttribute("hidden", "");
      }
    } else if (name === "value") {
      element.value = value;
      element.setAttribute("value", value);
    } else {
      element.setAttribute(name, value);
    }
  }
  parent.appendChild(element);
  return element;
}

function addNamedControl(parent, tagName, name, value = "") {
  return appendElement(parent, tagName, { name, value });
}

function buildDomFromIndex() {
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  assert.match(html, /<script type="module" src="\/app\.js"><\/script>/);

  const document = new MiniDocument();
  const byId = (id) => document.getElementById(id);
  const idTags = [...html.matchAll(/<([a-z]+)[^>]*\bid="([^"]+)"[^>]*>/gi)];

  for (const [, tagName, id] of idTags) {
    const source = html.match(new RegExp(`<${tagName}[^>]*\\bid="${id}"[^>]*>`, "i"))?.[0] ?? "";
    appendElement(document.body, tagName, {
      id,
      class: source.match(/\bclass="([^"]*)"/i)?.[1] ?? "",
      hidden: /\bhidden\b/i.test(source),
    });
  }

  addNamedControl(byId("login-form"), "input", "username", "sales");
  addNamedControl(byId("login-form"), "input", "password", "demo");
  addNamedControl(byId("order-form"), "input", "lineId");
  addNamedControl(byId("order-form"), "input", "customer", "ACME");
  addNamedControl(byId("order-form"), "input", "quantity", "2500");
  addNamedControl(byId("order-form"), "select", "priority", "low");
  addNamedControl(byId("order-form"), "input", "dueDate");
  addNamedControl(byId("order-form"), "textarea", "note");
  addNamedControl(byId("schedule-form"), "input", "lineId");
  addNamedControl(byId("schedule-form"), "input", "startDate");
  addNamedControl(byId("schedule-form"), "input", "reason");
  addNamedControl(byId("production-form"), "input", "orderId");
  addNamedControl(byId("production-form"), "input", "productionDate");
  addNamedControl(byId("production-form"), "input", "producedQuantity");

  for (const mode of ["pending", "scheduled", "all"]) {
    appendElement(byId("main-calendar-mode"), "button", { "data-calendar-mode": mode });
    appendElement(byId("preview-calendar-mode"), "button", { "data-preview-calendar-mode": mode });
  }
  for (const view of ["orders", "calendar", "actions"]) {
    appendElement(document.body, "button", {
      class: view === "actions" ? "mobile-tab scheduler-only" : "mobile-tab",
      "data-mobile-view": view,
    });
  }
  appendElement(byId("active-line-select"), "option", { value: "A" });
  appendElement(byId("active-line-select"), "option", { value: "B" });
  appendElement(byId("active-line-select"), "option", { value: "C" });
  appendElement(byId("active-line-select"), "option", { value: "D" });

  return document;
}

function installBrowserGlobals(document) {
  return installBrowserGlobalsWithFetch(document, () => {
    throw new Error("anonymous startup must not call fetch");
  });
}

function installBrowserGlobalsWithFetch(document, fetchImpl, initialStorage = {}) {
  const previous = new Map();
  const storage = new Map(Object.entries(initialStorage).map(([key, value]) => [key, String(value)]));
  const localStorage = {
    getItem: (key) => storage.has(key) ? storage.get(key) : null,
    setItem: (key, value) => storage.set(key, String(value)),
    removeItem: (key) => storage.delete(key),
    clear: () => storage.clear(),
  };
  const window = {
    document,
    localStorage,
    confirm: () => true,
    setInterval: () => 1,
    clearInterval: () => {},
    setTimeout: (callback) => {
      callback();
      return 1;
    },
    clearTimeout: () => {},
  };
  const globals = {
    document,
    window,
    localStorage,
    fetch: fetchImpl,
    FormData: MiniFormData,
    HTMLElement: MiniElement,
    HTMLDialogElement: MiniElement,
    CSS: { escape: (value) => String(value).replaceAll('"', '\\"') },
  };
  for (const [key, value] of Object.entries(globals)) {
    previous.set(key, globalThis[key]);
    globalThis[key] = value;
  }
  return () => {
    for (const [key, value] of previous) {
      if (value === undefined) {
        delete globalThis[key];
      } else {
        globalThis[key] = value;
      }
    }
  };
}

class MiniFormData {
  constructor(form) {
    this.entriesList = [];
    for (const child of walk(form)) {
      if (child.name) {
        this.entriesList.push([child.name, child.value]);
      }
    }
  }

  [Symbol.iterator]() {
    return this.entriesList[Symbol.iterator]();
  }
}

function jsonResponse(payload, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => payload,
  };
}

async function flushAsyncWork() {
  await Promise.resolve();
  await Promise.resolve();
  await Promise.resolve();
}

test("anonymous startup renders login state with fallback lines and initialized dates", async () => {
  const document = buildDomFromIndex();
  const restoreGlobals = installBrowserGlobals(document);
  try {
    await import(new URL(`./app.js?dp002=${Date.now()}`, import.meta.url));

    assert.equal(document.getElementById("login-page").hidden, false);
    assert.equal(document.getElementById("app-shell").hidden, true);
    assert.equal(document.body.dataset.role, "");
    assert.equal(document.getElementById("active-line-select").innerHTML.includes('value="A"'), true);
    assert.equal(document.getElementById("active-line-select").innerHTML.includes('value="D"'), true);
    assert.match(document.querySelector('input[name="startDate"]').value, /^\d{4}-\d{2}-\d{2}$/);
    assert.match(document.querySelector('input[name="dueDate"]').value, /^\d{4}-\d{2}-\d{2}$/);
    assert.notEqual(document.querySelector('input[name="startDate"]').value, "");
    assert.notEqual(document.querySelector('input[name="dueDate"]').value, "");
    assert.equal(document.querySelector('#order-form input[name="lineId"]').value, "A");
    assert.equal(document.querySelector('#schedule-form input[name="lineId"]').value, "A");
  } finally {
    restoreGlobals();
  }
});

test("login saves session and logout clears storage and app state", async () => {
  const document = buildDomFromIndex();
  const calls = [];
  const fetchImpl = async (path, options = {}) => {
    calls.push({ path, options });
    if (path === "/api/auth/login") {
      assert.equal(options.method, "POST");
      assert.deepEqual(JSON.parse(options.body), { username: "sales", password: "demo" });
      return jsonResponse({
        token: "token-sales",
        user: { username: "sales", role: "sales" },
      });
    }
    if (path === "/api/lines") {
      return jsonResponse({
        lines: [
          { id: "A", name: "Line A", capacityPerDay: 10000, timezone: "Asia/Taipei" },
          { id: "B", name: "Line B", capacityPerDay: 9000, timezone: "Asia/Taipei" },
        ],
      });
    }
    if (path === "/api/orders") {
      return jsonResponse({ orders: [] });
    }
    if (String(path).startsWith("/api/schedules/calendar?")) {
      return jsonResponse({ allocations: [], pendingAllocations: [] });
    }
    if (path === "/api/auth/logout") {
      assert.equal(options.method, "POST");
      assert.equal(options.headers.Authorization, "Bearer token-sales");
      return jsonResponse({});
    }
    throw new Error(`unexpected fetch ${path}`);
  };
  const restoreGlobals = installBrowserGlobalsWithFetch(document, fetchImpl);
  try {
    await import(new URL(`./app.js?dp003=${Date.now()}`, import.meta.url));

    await document.getElementById("login-form").dispatchEvent({ type: "submit" });

    assert.equal(calls[0].path, "/api/auth/login");
    assert.equal(localStorage.getItem("woms.token"), "token-sales");
    assert.equal(localStorage.getItem("woms.user"), JSON.stringify({ username: "sales", role: "sales" }));
    assert.equal(document.getElementById("login-page").hidden, true);
    assert.equal(document.getElementById("app-shell").hidden, false);
    assert.equal(document.body.dataset.role, "sales");
    assert.equal(document.getElementById("order-form").hidden, false);
    assert.equal(document.getElementById("session-greeting").textContent, "您好 sales");

    await document.getElementById("logout-button").dispatchEvent({ type: "click" });

    assert.equal(calls.at(-1).path, "/api/auth/logout");
    assert.equal(localStorage.getItem("woms.token"), null);
    assert.equal(localStorage.getItem("woms.user"), null);
    assert.equal(document.getElementById("login-page").hidden, false);
    assert.equal(document.getElementById("app-shell").hidden, true);
    assert.equal(document.body.dataset.role, "");
  } finally {
    restoreGlobals();
  }
});

test("expired stored session is cleared and returns to login state", async () => {
  const document = buildDomFromIndex();
  const calls = [];
  const fetchImpl = async (path, options = {}) => {
    calls.push({ path, options });
    assert.equal(options.headers.Authorization, "Bearer expired-token");
    if (path === "/api/lines") {
      return jsonResponse({ error: "session expired" }, 401);
    }
    throw new Error(`unexpected fetch ${path}`);
  };
  const restoreGlobals = installBrowserGlobalsWithFetch(document, fetchImpl, {
    "woms.token": "expired-token",
    "woms.user": JSON.stringify({ username: "sales", role: "sales" }),
  });
  try {
    await import(new URL(`./app.js?dp004=${Date.now()}`, import.meta.url));
    await flushAsyncWork();

    assert.equal(calls.length, 1);
    assert.equal(calls[0].path, "/api/lines");
    assert.equal(localStorage.getItem("woms.token"), null);
    assert.equal(localStorage.getItem("woms.user"), null);
    assert.equal(document.getElementById("login-page").hidden, false);
    assert.equal(document.getElementById("app-shell").hidden, true);
    assert.equal(document.body.dataset.role, "");
    assert.equal(document.getElementById("message-title").textContent, "登入狀態已失效");
    assert.equal(document.getElementById("message-body").textContent, "session expired");
    assert.equal(document.getElementById("message-dialog").dataset.type, "warn");
  } finally {
    restoreGlobals();
  }
});

test("sales draft flow validates future due dates previews and confirms through APIs", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const orderSubmitStart = app.indexOf('document.getElementById("order-form").addEventListener("submit"');
  const orderSubmitEnd = app.indexOf('document.getElementById("assign-user-form")', orderSubmitStart);
  const orderSubmit = app.slice(orderSubmitStart, orderSubmitEnd);
  const confirmStart = app.indexOf('document.getElementById("confirm-preview-order").addEventListener("click"');
  const confirmEnd = app.indexOf('document.getElementById("confirm-schedule-job")', confirmStart);
  const confirm = app.slice(confirmStart, confirmEnd);

  assert.match(html, /id="order-form"[\s\S]*name="customer"[\s\S]*name="quantity"[\s\S]*name="priority"[\s\S]*name="dueDate"/);
  assert.match(orderSubmit, /assertFutureDueDate\(draftOrder\.dueDate\)/);
  assert.match(orderSubmit, /await createPreview\(\{[\s\S]*draftOrder,[\s\S]*\}, "sales-draft"\)/);
  assert.match(app, /request\("\/api\/schedules\/preview"/);
  assert.match(confirm, /request\("\/api\/orders\/preview-confirm", \{[\s\S]*method: "POST"[\s\S]*previewId: state\.preview\.previewId/);
  assert.match(confirm, /focusCreatedOrder\(order\)/);
  assert.match(confirm, /await refreshWorkspace\(\)/);
  assert.match(app, /showMessage\("無法加入待排程", error\.message, "warn"\)/);
});

test("scheduler bulk controls filter select preview reject cancel and schedule selected orders", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const renderFiltersStart = app.indexOf("function renderFilters()");
  const renderFiltersEnd = app.indexOf("function roleLabel", renderFiltersStart);
  const filters = app.slice(renderFiltersStart, renderFiltersEnd);

  assert.match(html, /id="status-sidebar"/);
  assert.match(html, /id="customer-filter"/);
  assert.match(html, /id="priority-filters"/);
  assert.match(html, /id="selected-count"/);
  assert.match(html, /id="preview-selected"/);
  assert.match(html, /id="reject-selected"/);
  assert.match(html, /id="cancel-selected"/);
  assert.match(filters, /renderCustomerFilter\(\)/);
  assert.match(filters, /renderCheckboxGroup\("priority-filters", priorities, state\.filters\.priorities, priorityLabel\)/);
  assert.match(app, /state\.filters\.status = state\.filters\.status === status \? "" : status/);
  assert.match(app, /function updateSelectedCount\(\) \{[\s\S]*已選取 \$\{state\.selectedOrderIds\.size\} 張訂單/);
  assert.match(app, /document\.getElementById\("preview-selected"\)\.addEventListener\("click"[\s\S]*await createPreview\(data, "schedule"\)/);
  assert.match(app, /document\.getElementById\("reject-selected"\)\.addEventListener\("click"[\s\S]*openRejectDialog\(Array\.from\(state\.selectedOrderIds\)\)/);
  assert.match(app, /request\("\/api\/orders\/reject", \{[\s\S]*method: "POST"[\s\S]*orderIds: state\.rejectOrderIds/);
  assert.match(app, /document\.getElementById\("cancel-selected"\)\.addEventListener\("click"[\s\S]*request\("\/api\/orders", \{[\s\S]*method: "DELETE"[\s\S]*Array\.from\(state\.selectedOrderIds\)/);
  assert.match(app, /document\.getElementById\("schedule-form"\)\.addEventListener\("submit"[\s\S]*await createPreview\(data, "schedule"\)/);
});

test("calendar modes and drag or drop scheduling target visible future dates", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");

  assert.match(html, /id="main-calendar-mode"[\s\S]*data-calendar-mode="pending"[\s\S]*data-calendar-mode="scheduled"[\s\S]*data-calendar-mode="all"/);
  assert.match(app, /document\.getElementById\("main-calendar-mode"\)\.addEventListener\("click"[\s\S]*state\.calendarMode = mode[\s\S]*renderCalendar\(\)/);
  assert.match(app, /function mainCalendarAllocations\(\)[\s\S]*state\.calendarMode === "pending"[\s\S]*state\.calendarMode === "all"/);
  assert.match(app, /cell\.addEventListener\("drop", async \(event\) => \{[\s\S]*await scheduleDroppedOrders\(orderIds, day\.key\)/);
  assert.equal(app.includes('document.elementFromPoint(clientX, clientY)?.closest?.(".calendar-day")'), true);
  assert.match(app, /async function scheduleDroppedOrders\(orderIds, targetDate\) \{[\s\S]*startDate: targetDate,[\s\S]*orderIds,[\s\S]*manualForce: false/);
  assert.match(app, /request\("\/api\/schedules\/preview"/);
});

test("conflict preview actions retry edit unselect reject solve and validate manual force", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const previewHandlerStart = app.indexOf('document.getElementById("preview-page-list").addEventListener("click"');
  const previewHandlerEnd = app.indexOf('document.getElementById("prev-month")', previewHandlerStart);
  const previewHandler = app.slice(previewHandlerStart, previewHandlerEnd);

  assert.match(previewHandler, /action === "retry-today"/);
  assert.match(previewHandler, /action === "retry-suggested-start"/);
  assert.match(app, /data-preview-action="update-conflict-due-date"/);
  assert.match(app, /data-preview-action="unselect-conflict-order"/);
  assert.match(app, /data-preview-action="reject-preview-orders"/);
  assert.match(app, /data-preview-action="preview-conflict-solution"/);
  assert.match(previewHandler, /action === "retry-manual-force"/);
  assert.match(previewHandler, /await retryPreview\(\{ startDate: tomorrowDateInputValue\(\), manualForce: false, reason: "" \}\)/);
  assert.match(previewHandler, /await updateOrderDueDate\(orderId, input\.value\)/);
  assert.match(previewHandler, /state\.selectedOrderIds\.delete\(orderId\)/);
  assert.match(previewHandler, /openRejectDialog\(state\.preview\.request\.orderIds\)/);
  assert.match(previewHandler, /orderIds\.length === 0[\s\S]*至少選取一張訂單/);
  assert.match(previewHandler, /!reason[\s\S]*人工強制介入必須留下原因/);
  assert.match(previewHandler, /await retryPreview\(\{ manualForce: true, reason \}\)/);
});

test("production flow starts scheduled orders and confirms allocation quantities", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const productionStart = app.indexOf("async function handleOrderAction");
  const productionEnd = app.indexOf("async function scheduleDroppedOrders", productionStart);
  const productionAction = app.slice(productionStart, productionEnd);
  const submitStart = app.indexOf("async function submitProductionReport");
  const submitEnd = app.indexOf("function suggestedStartDate", submitStart);
  const submit = app.slice(submitStart, submitEnd);

  assert.match(app, /data-order-action="start-production"/);
  assert.match(app, /data-order-action="confirm-production"/);
  assert.match(productionAction, /request\("\/api\/production\/start", \{[\s\S]*method: "POST"[\s\S]*JSON\.stringify\(\{ orderId \}\)/);
  assert.match(productionAction, /openProductionReport\(order, productionDate\)/);
  assert.match(app, /document\.getElementById\("production-form"\)\.addEventListener\("submit"[\s\S]*submitProductionReport\(form\.orderId, form\.productionDate, Number\(form\.producedQuantity\)\)/);
  assert.match(submit, /producedQuantity <= 0/);
  assert.match(submit, /producedQuantity > allocation\.quantity/);
  assert.match(submit, /request\("\/api\/production\/confirm", \{[\s\S]*method: "POST"[\s\S]*orderId, productionDate, producedQuantity/);
  assert.match(submit, /payload\.remainder/);
});

test("admin user management and HPA browser controls call expected APIs", () => {
  const app = readFileSync(new URL("./app.js", import.meta.url), "utf8");
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");

  assert.match(html, /id="create-user-form"/);
  assert.match(html, /id="assign-user-form"/);
  assert.match(html, /id="reset-password-form"/);
  assert.match(html, /id="delete-user-button"/);
  assert.match(app, /document\.getElementById\("create-user-form"\)\.addEventListener\("submit"[\s\S]*request\("\/api\/users", \{[\s\S]*method: "POST"/);
  assert.match(app, /document\.getElementById\("assign-user-form"\)\.addEventListener\("submit"[\s\S]*request\("\/api\/users", \{[\s\S]*method: "PATCH"/);
  assert.match(app, /document\.getElementById\("reset-password-form"\)\.addEventListener\("submit"[\s\S]*request\("\/api\/users\/password", \{[\s\S]*method: "PATCH"/);
  assert.match(app, /request\(`\/api\/users\/\$\{encodeURIComponent\(username\)\}`, \{ method: "DELETE" \}\)/);
  assert.match(html, /id="create-hpa-peak"/);
  assert.match(html, /id="refresh-hpa-peak"/);
  assert.match(html, /id="clear-hpa-peak"/);
  assert.match(app, /request\("\/api\/demo\/hpa-peak", \{ method: "POST" \}\)/);
  assert.match(app, /request\("\/api\/demo\/hpa-peak"\)/);
  assert.match(app, /request\("\/api\/demo\/hpa-peak", \{ method: "DELETE" \}\)/);
  assert.match(app, /state\.hpaPeakPollingEnabled = true/);
  assert.match(app, /function syncHPAPeakPolling\(\)/);
});
