import { Generate, Status, Clear, OpenDataDir, OpenExternalLink } from '../wailsjs/go/main/App.js';

window.openExternalLink = OpenExternalLink;

const form = document.getElementById('certForm');
const btn = document.getElementById('generateBtn');
const resultSection = document.getElementById('resultSection');
const errorSection = document.getElementById('errorSection');
const lifetimeWarn = document.getElementById('lifetimeWarn');

function checkLifetime() {
  const cl = parseInt(form.querySelector('[name="cert_lifetime"]').value) || 0;
  lifetimeWarn.classList.toggle('hidden', cl <= 3650);
}
form.querySelector('[name="cert_lifetime"]').addEventListener('input', checkLifetime);
checkLifetime();

const dlIcon = '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>';

function showResult(d, savedInfo) {
  document.getElementById('caInfo').innerHTML =
    `<p>Subject: ${d.ca.subject}</p><p>有效期至: ${d.ca.not_after}</p>`;
  document.getElementById('serverInfo').innerHTML =
    `<p>Subject: ${d.server.subject}</p><p>有效期至: ${d.server.not_after}</p>`;
  if (d.server.cert_pem) document.getElementById('serverCertPem').textContent = d.server.cert_pem;
  if (d.server.key_pem) document.getElementById('serverKeyPem').textContent = d.server.key_pem;

  const clientDiv = document.getElementById('clientInfo');
  clientDiv.innerHTML = '';
  for (const c of (d.clients || [])) {
    clientDiv.innerHTML += `
      <div class="text-xs text-slate-400">
        <p class="font-medium text-slate-300">${c.name}</p>
        <p>Subject: ${c.subject} | 有效期至: ${c.not_after}</p>
        <p class="text-slate-500 mt-0.5">文件: client_${c.name}.p12 / clientCert_${c.name}.crt</p>
      </div>`;
  }

  if (savedInfo) {
    const infoDiv = document.getElementById('savedInfoSection');
    infoDiv.classList.remove('hidden');
    document.getElementById('savedDomain').textContent = savedInfo.domain || '';
    document.getElementById('savedCaPass').textContent = savedInfo.ca_pass || '';
    document.getElementById('savedClientPass').textContent = savedInfo.client_pass || '';
    document.getElementById('savedGenTime').textContent = savedInfo.generated_at || new Date().toLocaleString();
  }

  resultSection.classList.remove('hidden');
}

function hideResult() {
  resultSection.classList.add('hidden');
  document.getElementById('savedInfoSection').classList.add('hidden');
  document.getElementById('serverCertPem').textContent = '';
  document.getElementById('serverKeyPem').textContent = '';
}

function fillForm(d) {
  form.querySelector('[name="domain"]').value = d.domain || '';
  form.querySelector('[name="country"]').value = d.country || 'CN';
  form.querySelector('[name="org"]').value = d.org || 'IKEv2VPN';
  form.querySelector('[name="ca_name"]').value = d.ca_name || 'ikev2ca';
  form.querySelector('[name="shared_san"]').value = d.shared_san || 'IKEv2Clients';
  form.querySelector('[name="client_names"]').value = (d.client_names || []).join(' ');
  form.querySelector('[name="ca_lifetime"]').value = d.ca_lifetime || 3652;
  form.querySelector('[name="cert_lifetime"]').value = d.cert_lifetime || 18250;
  form.querySelector('[name="ca_pass"]').value = d.ca_pass || '';
  form.querySelector('[name="client_pass"]').value = d.client_pass || '';
}

async function loadExisting() {
  try {
    const data = await Status();
    if (!data.success || !data.data.exists) return;
    const d = data.data;
    fillForm(d);
    showResult({
      ca: {subject: d.ca_subject, not_after: d.ca_not_after},
      server: {subject: d.server_subject, not_after: d.server_not_after, cert_pem: d.server_cert_pem || null, key_pem: d.server_key_pem || null},
      clients: d.clients
    }, {domain: d.domain, ca_pass: d.ca_pass, client_pass: d.client_pass, generated_at: d.generated_at});
  } catch (e) {}
}

loadExisting();

document.getElementById('btnClear').addEventListener('click', async () => {
  if (!confirm('确定要清理所有已生成的证书吗？此操作不可恢复！')) return;
  try { await Clear(); } catch (e) {}
  hideResult();
  form.querySelector('[name="domain"]').value = '';
  form.querySelector('[name="ca_pass"]').value = '';
  form.querySelector('[name="client_pass"]').value = '';
  form.querySelector('[name="client_names"]').value = 'win android ios';
});

document.getElementById('btnOpenDir').addEventListener('click', async () => {
  try { await OpenDataDir(); } catch (e) {}
});

form.addEventListener('submit', async (e) => {
  e.preventDefault();
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner mr-2"></span>正在生成...';
  hideResult();
  errorSection.classList.add('hidden');

  const clientNames = form.querySelector('[name="client_names"]').value.split(/\s+/).filter(Boolean);
  const body = {
    country: form.querySelector('[name="country"]').value,
    org: form.querySelector('[name="org"]').value,
    ca_name: form.querySelector('[name="ca_name"]').value,
    domain: form.querySelector('[name="domain"]').value,
    shared_san: form.querySelector('[name="shared_san"]').value,
    client_names: clientNames,
    ca_lifetime: parseInt(form.querySelector('[name="ca_lifetime"]').value, 10),
    cert_lifetime: parseInt(form.querySelector('[name="cert_lifetime"]').value, 10),
    ca_pass: form.querySelector('[name="ca_pass"]').value,
    client_pass: form.querySelector('[name="client_pass"]').value,
  };

  try {
    const data = await Generate(body);
    if (!data.success) {
      document.getElementById('errorMsg').textContent = data.message || '未知错误';
      errorSection.classList.remove('hidden');
      return;
    }
    const d = data.data;
    showResult({
      ca: {subject: d.ca_subject, not_after: d.ca_not_after},
      server: {subject: d.server_subject, not_after: d.server_not_after, cert_pem: d.server_cert_pem, key_pem: d.server_key_pem},
      clients: d.clients
    }, {domain: body.domain, ca_pass: body.ca_pass, client_pass: body.client_pass, generated_at: null});
  } catch (err) {
    document.getElementById('errorMsg').textContent = '错误: ' + (err.message || String(err));
    errorSection.classList.remove('hidden');
  } finally {
    btn.disabled = false;
    btn.textContent = '生成证书';
  }
});
