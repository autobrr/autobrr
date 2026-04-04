import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import authEn from "./locales/en/auth.json";
import commonEn from "./locales/en/common.json";
import optionsEn from "./locales/en/options.json";
import settingsEn from "./locales/en/settings.json";
import filtersEn from "./locales/en/filters.json";
import authZhCN from "./locales/zh-CN/auth.json";
import commonZhCN from "./locales/zh-CN/common.json";
import optionsZhCN from "./locales/zh-CN/options.json";
import settingsZhCN from "./locales/zh-CN/settings.json";
import filtersZhCN from "./locales/zh-CN/filters.json";
import authEs from "./locales/es/auth.json";
import commonEs from "./locales/es/common.json";
import optionsEs from "./locales/es/options.json";
import settingsEs from "./locales/es/settings.json";
import filtersEs from "./locales/es/filters.json";

export const supportedLanguages = ["en", "zh-CN", "es"] as const;
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

  if (window.navigator.language.toLowerCase().startsWith("zh")) {
    return "zh-CN";
  }

  if (window.navigator.language.toLowerCase().startsWith("es")) {
    return "es";
  }

  return "en";
};

void i18n.use(initReactI18next).init({
  resources: {
    en: {
      common: commonEn,
      auth: authEn,
      options: optionsEn,
      settings: settingsEn,
      filters: filtersEn
    },
    "zh-CN": {
      common: commonZhCN,
      auth: authZhCN,
      options: optionsZhCN,
      settings: settingsZhCN,
      filters: filtersZhCN
    },
    es: {
      common: commonEs,
      auth: authEs,
      options: optionsEs,
      settings: settingsEs,
      filters: filtersEs
    }
  },
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
