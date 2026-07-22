/*
 * Injected into the page under test.
 *
 * Its only job is to turn an interaction into a durable target. A click at
 * (412, 380) is worthless as a test; getByRole('link', { name: 'About Us' })
 * is what somebody would have written by hand, and is what survives the next
 * redesign. Several strategies are reported at once because they age
 * differently, and the compiler picks the best one available.
 *
 * Recording happens in the capture phase and never prevents the default, so
 * the page behaves exactly as it would without us watching.
 */
(() => {
  if (window.__appladRecorder) return;
  window.__appladRecorder = true;

  const ROLE_BY_TAG = {
    a: 'link',
    button: 'button',
    input: 'textbox',
    textarea: 'textbox',
    select: 'combobox',
    img: 'img',
    h1: 'heading',
    h2: 'heading',
    h3: 'heading',
    h4: 'heading',
    nav: 'navigation',
  };

  const roleOf = (el) => {
    const explicit = el.getAttribute('role');
    if (explicit) return explicit;
    const tag = el.tagName.toLowerCase();
    if (tag === 'input') {
      const type = (el.getAttribute('type') || 'text').toLowerCase();
      if (type === 'submit' || type === 'button') return 'button';
      if (type === 'checkbox') return 'checkbox';
      if (type === 'radio') return 'radio';
      return 'textbox';
    }
    return ROLE_BY_TAG[tag] || '';
  };

  // The accessible name, near enough: what a screen reader would announce and
  // what Playwright matches on.
  const nameOf = (el) => {
    const aria = el.getAttribute('aria-label');
    if (aria) return aria.trim();
    const labelled = el.getAttribute('aria-labelledby');
    if (labelled) {
      const ref = document.getElementById(labelled);
      if (ref) return (ref.textContent || '').trim();
    }
    if (el.tagName === 'IMG') return (el.getAttribute('alt') || '').trim();
    if (el.tagName === 'INPUT' && el.labels && el.labels[0]) {
      return (el.labels[0].textContent || '').trim();
    }
    return (el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 80);
  };

  // A short, readable CSS path. Only a fallback, but it must always resolve.
  const cssOf = (el) => {
    if (el.id) return `#${CSS.escape(el.id)}`;
    const parts = [];
    let node = el;
    while (node && node.nodeType === 1 && parts.length < 4) {
      let part = node.tagName.toLowerCase();
      if (node.id) {
        parts.unshift(`#${CSS.escape(node.id)}`);
        break;
      }
      const parent = node.parentElement;
      if (parent) {
        const siblings = [...parent.children].filter((c) => c.tagName === node.tagName);
        if (siblings.length > 1) part += `:nth-of-type(${siblings.indexOf(node) + 1})`;
      }
      parts.unshift(part);
      node = node.parentElement;
    }
    return parts.join(' > ');
  };

  // Where this element sits among everything the chosen selector matches.
  // Without it a recording of the second phone number replays as the first,
  // or fails outright under strict matching.
  const indexOf = (el, target) => {
    let candidates = [];
    if (target.role && target.name) {
      candidates = [...document.querySelectorAll('*')].filter(
        (c) => roleOf(c) === target.role && nameOf(c) === target.name,
      );
    } else if (target.text) {
      candidates = [...document.querySelectorAll('*')].filter(
        (c) => c.children.length === 0 && (c.textContent || '').trim() === target.text,
      );
    }
    const i = candidates.indexOf(el);
    return i > 0 ? i : 0;
  };

  const targetFor = (el) => {
    const target = {
      testId: el.getAttribute('data-testid') || el.getAttribute('data-test-id') || '',
      role: roleOf(el),
      name: nameOf(el),
      label: el.tagName === 'INPUT' && el.labels && el.labels[0]
        ? (el.labels[0].textContent || '').trim()
        : '',
      placeholder: el.getAttribute('placeholder') || '',
      text: el.children.length === 0 ? (el.textContent || '').trim().slice(0, 80) : '',
      css: cssOf(el),
    };
    target.nth = indexOf(el, target);
    return target;
  };

  const describe = (kind, target, value) => {
    const what = target.name || target.text || target.placeholder || target.css;
    switch (kind) {
      case 'tap':
        return `tap ${what}`;
      case 'type':
        return `type ${JSON.stringify(value)} into ${what}`;
      case 'expectVisible':
        return `expect ${what} to be visible`;
      default:
        return kind;
    }
  };

  const emit = (kind, target, value) => {
    window.__appladStep(
      JSON.stringify({ kind, target, value: value || '', description: describe(kind, target, value) }),
    );
  };

  document.addEventListener(
    'click',
    (e) => {
      const el = e.target instanceof Element ? e.target : null;
      if (!el) return;
      // In assert mode a click marks something to check rather than something
      // to do, so the page is left alone.
      if (window.__appladAssertMode) {
        e.preventDefault();
        e.stopPropagation();
        emit('expectVisible', targetFor(el));
        return;
      }
      // Attribute the click to the interactive ancestor, which is what a
      // person means by "I clicked the link" when they hit the text inside it.
      const actionable = el.closest('a, button, [role="button"], input, select, textarea, label') || el;
      emit('tap', targetFor(actionable));
    },
    true,
  );

  // Typing is recorded once the field is done, not per keystroke.
  document.addEventListener(
    'change',
    (e) => {
      const el = e.target;
      if (!(el instanceof HTMLInputElement) && !(el instanceof HTMLTextAreaElement)) return;
      if (el.type === 'checkbox' || el.type === 'radio') return;
      emit('type', targetFor(el), el.value);
    },
    true,
  );

  document.addEventListener(
    'keydown',
    (e) => {
      if (e.key === 'Enter' && e.target instanceof Element) {
        emit('press', targetFor(e.target), 'Enter');
      }
    },
    true,
  );
})();
