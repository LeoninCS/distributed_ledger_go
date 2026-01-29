const displayJSON = (el, data) => {
  if (!el) return;
  if (typeof data === 'string') {
    el.textContent = data;
    return;
  }
  if (Array.isArray(data)) {
    el.textContent = data.map((item, idx) => `[#${idx + 1}]\n${JSON.stringify(item, null, 2)}`).join('\n\n');
    return;
  }
  if (data && typeof data === 'object') {
    el.innerHTML = Object.entries(data)
      .map(([key, value]) => {
        if (value && typeof value === 'object') {
          return `<span class="label">${key}:</span>\n${JSON.stringify(value, null, 2)}`;
        }
        return `<span class="label">${key}:</span>\n${value}`;
      })
      .join('\n');
    return;
  }
  el.textContent = JSON.stringify(data, null, 2);
};

const handleError = (el, err) => {
  console.error(err);
  displayJSON(el, err?.message || '请求失败');
};

const postJSON = async (path, body) => {
  const init = {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  };
  const res = await fetch(path, init);
  if (!res.ok) {
    const msg = await res.text();
    throw new Error(msg || res.statusText);
  }
  if (res.status === 204) return null;
  return res.json();
};

const views = document.querySelectorAll('.view');
const showView = (id) => {
  views.forEach((view) => {
    if (view.id === id) {
      view.removeAttribute('hidden');
    } else {
      view.setAttribute('hidden', 'hidden');
    }
  });
};

document.querySelectorAll('[data-target]').forEach((btn) => {
  btn.addEventListener('click', () => showView(btn.dataset.target));
});

document.querySelectorAll('[data-back]').forEach((btn) => {
  btn.addEventListener('click', () => showView('view-home'));
});

const bindRegisterButton = (btnId, resultId) => {
  const btn = document.getElementById(btnId);
  const result = document.getElementById(resultId);
  if (!btn) return;
  btn.addEventListener('click', async () => {
    btn.disabled = true;
    btn.textContent = '进行中';
    try {
      const data = await postJSON('/accounts/register');
      displayJSON(result, data);
    } catch (err) {
      handleError(result, err);
    } finally {
      btn.disabled = false;
      btn.textContent = '注册';
    }
  });
};

const bindAccountLookup = (formId, resultId) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const address = new FormData(form).get('address');
    if (!address) return;
    result.textContent = '查询中';
    try {
      const res = await fetch(`/accounts/${address}`);
      if (!res.ok) throw new Error(await res.text());
      displayJSON(result, await res.json());
    } catch (err) {
      handleError(result, err);
    }
  });
};

const bindMintForm = (formId, resultId) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const data = Object.fromEntries(new FormData(form).entries());
    result.textContent = '提交中';
    try {
      const payload = {
        sender: data.sender,
        receiver: data.receiver,
        amount: Number(data.amount),
        nonce: Number(data.nonce),
        private_key: data.key,
      };
      const res = await postJSON('/transactions/mint', payload);
      displayJSON(result, res);
    } catch (err) {
      handleError(result, err);
    }
  });
};

const bindTransferForm = (formId, resultId) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const data = Object.fromEntries(new FormData(form).entries());
    result.textContent = '发送中';
    try {
      const payload = {
        sender: data.sender,
        receiver: data.receiver,
        amount: Number(data.amount),
        nonce: Number(data.nonce),
        private_key: data.key,
      };
      const res = await postJSON('/transactions/transfer', payload);
      displayJSON(result, res);
    } catch (err) {
      handleError(result, err);
    }
  });
};

const bindRoleForm = (formId, resultId, endpoint) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const data = Object.fromEntries(new FormData(form).entries());
    result.textContent = '执行中';
    try {
      const payload = {
        creator_address: data.creator,
        target_address: data.target,
        private_key: data.key,
      };
      const res = await postJSON(endpoint, payload);
      displayJSON(result, res);
    } catch (err) {
      handleError(result, err);
    }
  });
};

const bindFreezeForm = (formId, resultId, endpoint) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const data = Object.fromEntries(new FormData(form).entries());
    result.textContent = '执行中';
    try {
      const payload = {
        sender: data.admin,
        receiver: data.target,
        amount: 0,
        nonce: 0,
        private_key: data.key,
      };
      const res = await postJSON(endpoint, payload);
      displayJSON(result, res);
    } catch (err) {
      handleError(result, err);
    }
  });
};

const bindQueryForm = (formId, resultId) => {
  const form = document.getElementById(formId);
  const result = document.getElementById(resultId);
  if (!form) return;
  form.addEventListener('submit', async (evt) => {
    evt.preventDefault();
    const data = Object.fromEntries(new FormData(form).entries());
    result.textContent = '查询中';
    try {
      const payload = {
        requester_address: data.address,
        private_key: data.key,
      };
      const res = await postJSON('/transactions/query', payload);
      displayJSON(result, res);
    } catch (err) {
      handleError(result, err);
    }
  });
};

bindRegisterButton('system-register-btn', 'system-register-result');
bindMintForm('founder-mint-form', 'founder-mint-result');
bindRoleForm('founder-promote-form', 'founder-promote-result', '/accounts/promote');
bindRoleForm('founder-demote-form', 'founder-demote-result', '/accounts/demote');
bindQueryForm('founder-query-form', 'founder-query-result');

bindAccountLookup('admin-account-form', 'admin-account-result');
bindTransferForm('admin-transfer-form', 'admin-transfer-result');
bindFreezeForm('admin-freeze-form', 'admin-freeze-result', '/transactions/freeze');
bindFreezeForm('admin-unfreeze-form', 'admin-unfreeze-result', '/transactions/unfreeze');
bindQueryForm('admin-query-form', 'admin-query-result');

bindAccountLookup('user-account-form', 'user-account-result');
bindTransferForm('user-transfer-form', 'user-transfer-result');
bindQueryForm('user-query-form', 'user-query-result');

const statusBtn = document.getElementById('refresh-status');
const statusView = document.getElementById('raft-status');
const refreshStatus = async () => {
  if (!statusView) return;
  statusView.textContent = '刷新中';
  try {
    const res = await fetch('/raft/status');
    if (!res.ok) throw new Error(await res.text());
    statusView.textContent = JSON.stringify(await res.json(), null, 2);
  } catch (err) {
    statusView.textContent = err?.message || '状态异常';
  }
};
if (statusBtn) {
  statusBtn.addEventListener('click', refreshStatus);
  refreshStatus();
}

const themeToggle = document.getElementById('theme-toggle');
const applyTheme = (mode) => {
  document.body.classList.toggle('dark', mode === 'dark');
  localStorage.setItem('ledger_theme', mode);
};
const savedTheme = localStorage.getItem('ledger_theme') || 'light';
applyTheme(savedTheme);
if (themeToggle) {
  themeToggle.addEventListener('click', () => {
    const current = document.body.classList.contains('dark') ? 'dark' : 'light';
    applyTheme(current === 'dark' ? 'light' : 'dark');
  });
}

showView('view-home');
