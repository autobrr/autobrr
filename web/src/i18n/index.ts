import i18n, { type BackendModule, type ResourceKey } from "i18next";
import { initReactI18next } from "react-i18next";

import authEn from "./locales/en/auth.json";
import commonEn from "./locales/en/common.json";
import optionsEn from "./locales/en/options.json";
import settingsEn from "./locales/en/settings.json";
import filtersEn from "./locales/en/filters.json";

// Only English is bundled; every other locale is fetched on demand as its own chunk.
const localeModules = import.meta.glob<{ default: ResourceKey }>([
  "./locales/*/*.json",
  "!./locales/en/*.json"
]);

const lazyLocaleBackend: BackendModule = {
  type: "backend",
  init() {},
  read(language, namespace, callback) {
    const load = localeModules[`./locales/${language}/${namespace}.json`];
    if (!load) {
      callback(null, {});
      return;
    }

    load()
      .then((module) => callback(null, module.default))
      .catch((err: unknown) => callback(err as Error, false));
  }
};

export const supportedLanguages = ["en", "de", "cs", "es", "fr", "ru", "no", "zh-CN"] as const;
export type Language = (typeof supportedLanguages)[number];

export const getInitialLanguage = (): Language => {
  if (typeof window === "undefined") {
    return "en";
  }

  const storage = localStorage.getItem("autobrr_settings");
  if (storage) {
    try {
      const json = JSON.parse(storage) as { language?: string };
      if (json.language && supportedLanguages.includes(json.language as Language)) {
        return json.language as Language;
      }
    } catch {
      // ignore invalid stored settings
    }
  }

  const lang = window.navigator.language.toLowerCase();
  if (lang.startsWith("zh")) {
    return "zh-CN";
  }
  if (lang.startsWith("fr")) {
    return "fr";
  }

  if (lang.startsWith("ru")) {
    return "ru";
  }

  if (lang.startsWith("no")) {
    return "no";
  }

  if (lang.startsWith("de")) {
    return "de";
  }

  if (lang.startsWith("es")) {
    return "es";
  }

  if (lang.startsWith("cs")) {
    return "cs";
  }

  return "en";
};

void i18n.use(initReactI18next).use(lazyLocaleBackend).init({
  resources: {
    en: {
      common: commonEn,
      auth: authEn,
      options: optionsEn,
      settings: settingsEn,
      filters: filtersEn
    }
  },
  partialBundledLanguages: true,
  lng: getInitialLanguage(),
  fallbackLng: "en",
  supportedLngs: supportedLanguages,
  defaultNS: "common",
  ns: ["common", "auth", "settings", "options", "filters"],
  interpolation: {
    escapeValue: false
  },
  react: {
    useSuspense: false
  }
});

export default i18n;
