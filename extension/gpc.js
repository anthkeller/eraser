(() => {
  try {
    if (navigator.globalPrivacyControl === true) return;
    Object.defineProperty(Navigator.prototype, "globalPrivacyControl", {
      configurable: true,
      enumerable: true,
      get: () => true
    });
  } catch (_) {
    // A browser-native implementation takes precedence if it cannot be changed.
  }
})();
