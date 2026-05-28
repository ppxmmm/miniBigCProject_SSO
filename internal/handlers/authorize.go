package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type loginData struct {
	AppName             string
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Error               string
}

const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Mini BigC SSO</title>
  <style>
    :root {
      --bg: #fbfbf7;
      --surface: #ffffff;
      --text: #27251f;
      --muted: #6c6a61;
      --border: #e4e1d8;
      --green: #16864a;
      --green-strong: #0f6b3b;
      --green-dark: #0d4f31;
      --green-soft: #eaf7ef;
      --yellow: #f5c84c;
      --danger: #b42318;
      --danger-bg: #fff3f1;
      --danger-border: #ffd1cb;
      --shadow: 0 24px 70px rgba(33, 39, 30, 0.12), 0 2px 8px rgba(33, 39, 30, 0.06);
    }

    *, *::before, *::after { box-sizing: border-box; }

    html { min-height: 100%; }

    body {
      margin: 0;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color: var(--text);
      background: #fff;
      min-height: 100vh;
      -webkit-font-smoothing: antialiased;
      text-rendering: optimizeLegibility;
    }

    .shell {
      position: relative;
      isolation: isolate;
      width: 100%;
      min-height: 100vh;
      display: block;
      overflow: hidden;
      background: #155b31;
    }

    .shell::before {
      content: "";
      position: absolute;
      inset: 0;
      z-index: -2;
      background: url("/assets/sso-store-woman-logo.png") center center / cover no-repeat;
      animation: imageSettle 1.1s cubic-bezier(0.22, 1, 0.36, 1) both;
    }

    .shell::after {
      content: "";
      position: absolute;
      inset: 0;
      z-index: -1;
      background:
        radial-gradient(ellipse at 72% 34%, rgba(16, 96, 48, 0.02) 0%, rgba(16, 96, 48, 0.08) 22%, rgba(16, 96, 48, 0.28) 44%, rgba(16, 96, 48, 0.68) 78%),
        linear-gradient(90deg, rgba(21, 91, 49, 0.92) 0%, rgba(21, 91, 49, 0.78) 36%, rgba(21, 91, 49, 0.42) 64%, rgba(21, 91, 49, 0.18) 100%),
        linear-gradient(180deg, rgba(21, 91, 49, 0.74) 0%, rgba(21, 91, 49, 0.24) 45%, rgba(21, 91, 49, 0.68) 100%);
    }

    .brand-panel {
      position: absolute;
      inset: 0;
      z-index: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      gap: 0;
      min-height: 100vh;
      padding: clamp(40px, 4.6vw, 72px);
      color: #fff;
      background: transparent;
      pointer-events: none;
    }

    .brand-panel::before {
      content: none;
    }

    .brand-panel::after {
      content: none;
    }

    .brand-row {
      display: flex;
      align-items: center;
      gap: 12px;
      animation: riseIn 0.7s cubic-bezier(0.22, 1, 0.36, 1) both;
    }

    .brand-mark {
      width: 64px;
      height: 64px;
      overflow: hidden;
      display: grid;
      place-items: center;
      border-radius: 7px;
      background: #fff;
      color: var(--green);
      font-size: 18px;
      font-weight: 800;
      letter-spacing: -0.04em;
      box-shadow: 0 12px 28px rgba(5, 31, 17, 0.18);
    }

    .brand-mark img {
      display: block;
      width: 100%;
      height: 100%;
      object-fit: contain;
      padding: 0;
    }

    .brand-title {
      margin: 0;
      font-size: 30px;
      line-height: 1.1;
      font-weight: 700;
      letter-spacing: -0.02em;
    }

    .brand-subtitle {
      margin: 6px 0 0;
      font-size: 18px;
      line-height: 1.45;
      color: rgba(255, 255, 255, 0.88);
    }

    .hero-copy {
      max-width: 560px;
      margin-top: auto;
      margin-bottom: clamp(24px, 5vh, 64px);
      animation: riseIn 0.75s 0.12s cubic-bezier(0.22, 1, 0.36, 1) both;
    }

    .hero-copy h1 {
      margin: 0;
      max-width: 560px;
      font-size: clamp(40px, 3.8vw, 66px);
      line-height: 1.13;
      font-weight: 720;
      letter-spacing: -0.035em;
    }

    .hero-copy p {
      margin: 22px 0 0;
      max-width: 560px;
      font-size: 18px;
      line-height: 1.5;
      color: rgba(255, 255, 255, 0.88);
    }

    .operations {
      width: min(100%, 900px);
      margin-top: 36px;
      margin-bottom: 0;
      animation: riseIn 0.75s 0.26s cubic-bezier(0.22, 1, 0.36, 1) both;
    }

    .signal-list {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .signal {
      display: flex;
      align-items: center;
      gap: 14px;
      min-height: 76px;
      padding: 13px 14px;
      border: 1px solid rgba(255, 255, 255, 0.15);
      border-radius: 8px;
      background: rgba(255, 255, 255, 0.10);
      backdrop-filter: blur(14px);
      transition: background 0.18s ease, transform 0.18s ease;
    }

    .signal:last-child {
      border-bottom: 0;
    }

    .signal:hover {
      background: rgba(255, 255, 255, 0.15);
      transform: translateY(-2px);
    }

    .signal-icon {
      width: 48px;
      height: 48px;
      display: grid;
      place-items: center;
      border-radius: 50%;
      background: #fff;
      color: #18914c;
      font-size: 20px;
      font-weight: 800;
      flex: 0 0 auto;
    }

    .signal-icon svg {
      width: 24px;
      height: 24px;
      stroke: currentColor;
    }

    .signal strong {
      display: block;
      font-size: 14px;
      line-height: 1.2;
      font-weight: 650;
    }

    .signal-text {
      display: block;
    }

    .signal-text span {
      display: block;
      margin-top: 5px;
      font-size: 12px;
      line-height: 1.35;
      color: rgba(255, 255, 255, 0.88);
    }

    .form-panel {
      position: relative;
      z-index: 2;
      display: flex;
      align-items: center;
      justify-content: flex-end;
      min-height: 100vh;
      padding: clamp(32px, 4vw, 64px) clamp(56px, 8vw, 148px);
      background: transparent;
    }

    .card {
      width: 100%;
      max-width: 440px;
      padding: 44px;
      border: 1px solid rgba(255, 255, 255, 0.78);
      border-radius: 8px;
      background: rgba(255, 255, 255, 0.96);
      box-shadow: 0 34px 100px rgba(9, 42, 24, 0.34), 0 2px 14px rgba(9, 42, 24, 0.10);
      backdrop-filter: blur(20px);
      animation: cardPop 0.82s 0.18s cubic-bezier(0.22, 1, 0.36, 1) both;
    }

    .mobile-brand {
      display: none;
      margin-bottom: 30px;
    }

    .form-header {
      margin-bottom: 42px;
      text-align: center;
    }

    .form-logo {
      width: 80px;
      height: 80px;
      display: block;
      margin: 0 auto 34px;
      object-fit: contain;
      border-radius: 8px;
    }

    .logo {
      margin: 0;
      font-size: 32px;
      line-height: 1;
      font-weight: 760;
      color: var(--text);
      letter-spacing: -0.035em;
    }

    .subtitle {
      margin: 18px 0 0;
      color: #7a8291;
      font-size: 16px;
      line-height: 1.55;
    }

    .subtitle strong { color: var(--green-dark); font-weight: 700; }

    .error {
      display: flex;
      align-items: flex-start;
      gap: 8px;
      background: var(--danger-bg);
      border: 1px solid var(--danger-border);
      color: var(--danger);
      padding: 10px 12px;
      border-radius: 8px;
      margin: 22px 0 0;
      font-size: 13px;
      line-height: 1.45;
      font-weight: 600;
    }

    .error::before {
      content: "!";
      width: 18px;
      height: 18px;
      flex: 0 0 18px;
      display: grid;
      place-items: center;
      border-radius: 50%;
      background: rgba(180, 35, 24, 0.12);
      font-size: 12px;
      font-weight: 800;
    }

    form {
      display: grid;
      gap: 24px;
    }

    .field { display: grid; gap: 10px; }

    .control {
      position: relative;
    }

    label {
      display: block;
      font-size: 14px;
      line-height: 1.3;
      font-weight: 650;
      color: #30332d;
    }

    input[type=email], input[type=password], input[type=text] {
      position: relative;
      z-index: 1;
      width: 100%;
      height: 48px;
      padding: 0 44px 0 48px;
      border: 1px solid #cfd5df;
      border-radius: 5px;
      background: #fff;
      color: var(--text);
      font-size: 16px;
      line-height: 1.3;
      outline: none;
      transition: border-color 0.16s ease, box-shadow 0.16s ease, background 0.16s ease, transform 0.16s ease;
    }

    input::placeholder { color: #8a93a3; }
    input:hover { border-color: #c8c4b9; }
    input:focus {
      border-color: var(--green);
      box-shadow: 0 0 0 3px rgba(22, 134, 74, 0.15);
      transform: translateY(-1px);
    }

    .input-icon,
    .input-action {
      position: absolute;
      top: 50%;
      width: 20px;
      height: 20px;
      color: #7f8794;
      transform: translateY(-50%);
      pointer-events: none;
      z-index: 2;
    }

    .input-icon { left: 17px; }
    .input-action { right: 17px; }

    .password-toggle {
      position: absolute;
      top: 50%;
      right: 10px;
      width: 36px;
      height: 36px;
      padding: 0;
      display: grid;
      place-items: center;
      border: 0;
      border-radius: 8px;
      background: transparent;
      color: #7f8794;
      box-shadow: none;
      transform: translateY(-50%);
      cursor: pointer;
      z-index: 3;
    }

    .password-toggle:hover {
      background: rgba(22, 134, 74, 0.08);
      color: var(--green);
      box-shadow: none;
      transform: translateY(-50%);
    }

    .password-toggle:focus-visible {
      outline: 3px solid rgba(22, 134, 74, 0.22);
      outline-offset: 2px;
    }

    .password-toggle svg {
      width: 20px;
      height: 20px;
    }

    .password-toggle .eye-open {
      display: none;
    }

    .password-toggle[aria-pressed="true"] .eye-open {
      display: block;
    }

    .password-toggle[aria-pressed="true"] .eye-closed {
      display: none;
    }

    .forgot-row {
      margin-top: -5px;
      text-align: right;
    }

    .forgot-link {
      color: var(--green);
      font-size: 14px;
      font-weight: 500;
      text-decoration: none;
    }

    .forgot-link:hover { text-decoration: underline; }
    .forgot-link:focus-visible {
      outline: 3px solid rgba(22, 134, 74, 0.22);
      outline-offset: 3px;
      border-radius: 4px;
    }

    button[type=submit] {
      width: 100%;
      height: 54px;
      padding: 0 18px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 0;
      background: var(--green);
      color: #fff;
      border: none;
      border-radius: 5px;
      font-size: 18px;
      line-height: 1;
      font-weight: 700;
      cursor: pointer;
      box-shadow: none;
      transition: background 0.16s ease, transform 0.16s ease, box-shadow 0.16s ease;
    }

    button[type=submit]:hover {
      background: var(--green-strong);
      transform: translateY(-1px);
      box-shadow: 0 12px 28px rgba(22, 134, 74, 0.18);
    }

    button[type=submit]:focus-visible {
      outline: 3px solid rgba(22, 134, 74, 0.24);
      outline-offset: 3px;
    }

    .button-arrow {
      display: none;
      font-size: 16px;
      line-height: 1;
    }

    .divider {
      height: 1px;
      margin: 36px 0 24px;
      background: #e2e5ea;
    }

    .trust-row {
      margin-top: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      color: #7a8291;
      font-size: 16px;
      line-height: 1.4;
      animation: fadeIn 0.65s 0.48s ease both;
    }

    .trust-icon {
      width: 21px;
      height: 21px;
      display: grid;
      place-items: center;
      color: #7f8794;
      background: transparent;
      font-size: 12px;
      font-weight: 800;
    }

    .footer {
      display: none;
    }

    .footer strong {
      color: var(--text);
      font-weight: 700;
    }

    @media (max-width: 880px) {
      body {
        display: block;
        padding: 0;
        background: var(--bg);
      }

      .shell {
        min-height: 100vh;
        display: block;
        border: 0;
        border-radius: 0;
        box-shadow: none;
        background: #fff;
      }

      .shell::before,
      .shell::after {
        content: none;
      }

      .brand-panel { display: none; }

      .form-panel {
        min-height: 100vh;
        align-items: flex-start;
        justify-content: center;
        padding: 32px 22px;
        background: #fff;
      }

      .card {
        max-width: 460px;
        padding: 0;
        border: 0;
        background: transparent;
        box-shadow: none;
        backdrop-filter: none;
      }

      .mobile-brand {
        display: flex;
        align-items: center;
        gap: 12px;
      }

      .form-header {
        margin-top: 28px;
        margin-bottom: 36px;
        text-align: left;
      }

      .form-logo { display: none; }

      .logo { font-size: 30px; }
    }

    @media (max-width: 1120px) {
      .brand-panel {
        position: relative;
        min-height: auto;
        padding-bottom: 0;
      }

      .shell {
        display: grid;
      }

      .form-panel {
        min-height: auto;
        justify-content: center;
        padding-top: clamp(32px, 6vw, 72px);
      }

      .signal-list { grid-template-columns: 1fr; max-width: 390px; }
    }

    @keyframes imageSettle {
      from {
        opacity: 0;
        transform: scale(1.035);
        filter: saturate(0.9);
      }
      to {
        opacity: 1;
        transform: scale(1);
        filter: saturate(1);
      }
    }

    @keyframes riseIn {
      from {
        opacity: 0;
        transform: translateY(18px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    @keyframes cardPop {
      from {
        opacity: 0;
        transform: translateY(18px) scale(0.985);
      }
      to {
        opacity: 1;
        transform: translateY(0) scale(1);
      }
    }

    @keyframes fadeIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }

    @media (prefers-reduced-motion: reduce) {
      *, *::before, *::after {
        animation-duration: 0.001ms !important;
        animation-iteration-count: 1 !important;
        scroll-behavior: auto !important;
        transition-duration: 0.001ms !important;
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="brand-panel" aria-label="Mini BigC operations">
      <div class="brand-row">
        <div class="brand-mark" aria-hidden="true"><img src="/assets/brand-logo.ico" alt=""></div>
        <div>
          <p class="brand-title">Mini BigC</p>
          <p class="brand-subtitle">Manager Console</p>
        </div>
      </div>

      <div class="hero-copy">
        <h1>Run your branch with confidence</h1>
        <p>Real-time insights and tools to help your store operate efficiently every day.</p>
      </div>

      <div class="operations">
        <ul class="signal-list" aria-label="Console capabilities">
          <li class="signal">
            <span class="signal-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M4 19h16" stroke-width="1.9" stroke-linecap="round"/>
                <path d="M7 16V9M12 16V5M17 16v-4" stroke-width="1.9" stroke-linecap="round"/>
                <path d="m15 7 2-2 2 2" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </span>
            <span class="signal-text"><strong>Real-time sales</strong><span>Monitor performance and trends as they happen.</span></span>
          </li>
          <li class="signal">
            <span class="signal-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M6.5 10.5a5.5 5.5 0 0 1 11 0c0 4 1.6 5.4 2.3 6H4.2c.7-.6 2.3-2 2.3-6Z" stroke-width="1.9" stroke-linejoin="round"/>
                <path d="M10 19a2.2 2.2 0 0 0 4 0" stroke-width="1.9" stroke-linecap="round"/>
              </svg>
            </span>
            <span class="signal-text"><strong>Stock alerts</strong><span>Stay ahead with smart stock level notifications.</span></span>
          </li>
          <li class="signal">
            <span class="signal-icon" aria-hidden="true">
              <svg viewBox="0 0 24 24" fill="none">
                <path d="M3.5 7h10v9h-10zM13.5 10h3.5l3.5 3.2V16h-7" stroke-width="1.9" stroke-linejoin="round"/>
                <path d="M7 18.5a1.8 1.8 0 1 0 0-3.6 1.8 1.8 0 0 0 0 3.6ZM17.8 18.5a1.8 1.8 0 1 0 0-3.6 1.8 1.8 0 0 0 0 3.6Z" stroke-width="1.9"/>
              </svg>
            </span>
            <span class="signal-text"><strong>Delivery tracking</strong><span>Track deliveries and ensure timely arrivals.</span></span>
          </li>
        </ul>
      </div>
    </section>

    <main class="form-panel">
      <div class="card">
        <div class="mobile-brand">
          <div class="brand-mark" aria-hidden="true"><img src="/assets/brand-logo.ico" alt=""></div>
          <div>
            <p class="brand-title" style="color: var(--text)">Mini BigC</p>
            <p class="brand-subtitle" style="color: var(--muted)">Manager Console</p>
          </div>
        </div>

        <div class="form-header">
          <img class="form-logo" src="/assets/brand-logo.ico" alt="Mini BigC">
          <h2 class="logo">Mini BigC SSO</h2>
          {{if .AppName}}
          <p class="subtitle">Sign in to continue to {{.AppName}}</p>
          {{else}}
          <p class="subtitle">Sign in with your staff account.</p>
          {{end}}
        </div>
        {{if .Error}}<div class="error" role="alert">{{.Error}}</div>{{end}}

        <form method="POST" action="/authorize">
          <input type="hidden" name="client_id"             value="{{.ClientID}}">
          <input type="hidden" name="redirect_uri"          value="{{.RedirectURI}}">
          <input type="hidden" name="response_type"         value="{{.ResponseType}}">
          <input type="hidden" name="scope"                 value="{{.Scope}}">
          <input type="hidden" name="state"                 value="{{.State}}">
          <input type="hidden" name="nonce"                 value="{{.Nonce}}">
          <input type="hidden" name="code_challenge"        value="{{.CodeChallenge}}">
          <input type="hidden" name="code_challenge_method" value="{{.CodeChallengeMethod}}">

          <div class="field">
            <label for="email">Email</label>
            <div class="control">
              <svg class="input-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none">
                <path d="M4 6.5h16v11H4z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
                <path d="m5.5 8 6.5 5 6.5-5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              <input type="email" id="email" name="email" required autofocus autocomplete="email" inputmode="email" placeholder="Enter your email">
            </div>
          </div>

          <div class="field">
            <label for="password">Password</label>
            <div class="control">
              <svg class="input-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none">
                <path d="M7 10V8a5 5 0 0 1 10 0v2" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
                <path d="M5.5 10h13v9h-13z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
              </svg>
              <input type="password" id="password" name="password" required autocomplete="current-password" placeholder="Enter your password">
              <button class="password-toggle" type="button" aria-label="Show password" aria-controls="password" aria-pressed="false">
                <svg class="eye-closed" aria-hidden="true" viewBox="0 0 24 24" fill="none">
                  <path d="M3 3l18 18" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
                  <path d="M10.6 10.6a2 2 0 0 0 2.8 2.8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>
                  <path d="M9.3 5.6A8.8 8.8 0 0 1 12 5c4.5 0 8 4 9 7-.4 1.2-1.2 2.5-2.3 3.6M6.4 6.8C4.8 8 3.6 9.9 3 12c1 3 4.5 7 9 7 1.4 0 2.7-.4 3.8-1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                <svg class="eye-open" aria-hidden="true" viewBox="0 0 24 24" fill="none">
                  <path d="M3 12s3.4-6 9-6 9 6 9 6-3.4 6-9 6-9-6-9-6Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
                  <path d="M12 14.8a2.8 2.8 0 1 0 0-5.6 2.8 2.8 0 0 0 0 5.6Z" stroke="currentColor" stroke-width="1.8"/>
                </svg>
              </button>
            </div>
          </div>

          <div class="forgot-row">
            <a class="forgot-link" href="#" aria-label="Forgot your password?">Forgot your password?</a>
          </div>

          <button type="submit">Sign in<span class="button-arrow" aria-hidden="true">→</span></button>
        </form>

        <div class="divider" aria-hidden="true"></div>
        <div class="trust-row">
          <svg class="trust-icon" aria-hidden="true" viewBox="0 0 24 24" fill="none">
            <path d="M12 3.5 19 6v5.2c0 4.2-2.8 7.4-7 9.3-4.2-1.9-7-5.1-7-9.3V6l7-2.5Z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"/>
            <path d="m9 12 2 2 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <span>Protected by OAuth2 / OIDC</span>
        </div>

        <p class="footer"><strong>Secure access for Mini BigC staff.</strong><br>Use the same account issued for branch operations.</p>
      </div>
    </main>
  </div>
  <script>
    (() => {
      const password = document.getElementById("password");
      const toggle = document.querySelector(".password-toggle");
      if (!password || !toggle) return;

      toggle.addEventListener("click", () => {
        const show = password.type === "password";
        password.type = show ? "text" : "password";
        toggle.setAttribute("aria-pressed", String(show));
        toggle.setAttribute("aria-label", show ? "Hide password" : "Show password");
      });
    })();
  </script>
</body>
</html>`

var loginTmpl = template.Must(template.New("login").Parse(loginHTML))

func (h *Handler) AuthorizeGET(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")

	if clientID == "" || redirectURI == "" || responseType == "" {
		http.Error(w, "missing required parameters: client_id, redirect_uri, response_type", http.StatusBadRequest)
		return
	}
	if responseType != "code" {
		http.Error(w, "unsupported response_type: only 'code' is supported", http.StatusBadRequest)
		return
	}

	client, err := h.db.GetClientByClientID(clientID)
	if err != nil {
		log.Printf("authorize: get client: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if client == nil {
		http.Error(w, "unknown client_id", http.StatusBadRequest)
		return
	}
	if !containsURI(client.RedirectURIs, redirectURI) {
		http.Error(w, "redirect_uri not registered for this client", http.StatusBadRequest)
		return
	}

	data := loginData{
		AppName:             client.Name,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        responseType,
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTmpl.Execute(w, data)
}

func (h *Handler) AuthorizePOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	scope := r.FormValue("scope")
	state := r.FormValue("state")
	nonce := r.FormValue("nonce")
	challenge := r.FormValue("code_challenge")
	challengeMethod := r.FormValue("code_challenge_method")
	email := r.FormValue("email")
	password := r.FormValue("password")

	client, err := h.db.GetClientByClientID(clientID)
	if err != nil || client == nil || !containsURI(client.RedirectURIs, redirectURI) {
		http.Error(w, "invalid client", http.StatusBadRequest)
		return
	}

	baseData := loginData{
		AppName:             client.Name,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		ResponseType:        "code",
		Scope:               scope,
		State:               state,
		Nonce:               nonce,
		CodeChallenge:       challenge,
		CodeChallengeMethod: challengeMethod,
	}

	user, err := h.db.GetUserByEmail(email)
	if err != nil {
		log.Printf("authorize: get user: %v", err)
		h.renderLoginError(w, baseData, "An internal error occurred. Please try again.")
		return
	}
	if user == nil || !checkPassword(user.PasswordHash, password) {
		h.renderLoginError(w, baseData, "Invalid email or password.")
		return
	}

	finalScope := intersectScope(scope, client.Scopes)
	if !strings.Contains(finalScope, "openid") {
		finalScope = strings.TrimSpace("openid " + finalScope)
	}

	code, err := h.db.CreateAuthCode(clientID, user.ID, redirectURI, finalScope, nonce, challenge, challengeMethod)
	if err != nil {
		log.Printf("authorize: create auth code: %v", err)
		h.renderLoginError(w, baseData, "An internal error occurred. Please try again.")
		return
	}

	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code.Code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (h *Handler) renderLoginError(w http.ResponseWriter, data loginData, msg string) {
	data.Error = msg
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	loginTmpl.Execute(w, data)
}

func containsURI(uris []string, uri string) bool {
	for _, u := range uris {
		if u == uri {
			return true
		}
	}
	return false
}

func intersectScope(requested, allowed string) string {
	allowedSet := make(map[string]bool)
	for _, s := range strings.Fields(allowed) {
		allowedSet[s] = true
	}
	var result []string
	for _, s := range strings.Fields(requested) {
		if allowedSet[s] {
			result = append(result, s)
		}
	}
	return strings.Join(result, " ")
}
