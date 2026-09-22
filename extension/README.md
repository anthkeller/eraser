# Eraser Privacy Control extension

This Manifest V3 companion sends `Sec-GPC: 1` and `DNT: 1` with HTTP and HTTPS
requests. It also exposes `navigator.globalPrivacyControl` as `true` at
`document_start` so sites can observe the preference through JavaScript.

The popup does not record browsing history. A domain is stored in extension
local storage only after the user selects **Remember this account site**.

## Load for development

1. Open `chrome://extensions` in Chrome, Edge, Brave, or another Chromium browser.
2. Enable Developer mode.
3. Select **Load unpacked**.
4. Select this `extension` directory.
5. Visit the GPC reference test site and confirm both the HTTP and JavaScript signals.

`DNT: 1` is included as a legacy preference signal. GPC is the primary universal
opt-out mechanism. This extension expresses a preference; it does not claim that
every visited business is legally required to honor it.
