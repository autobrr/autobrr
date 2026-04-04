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
import authDe from "./locales/de/auth.json";
import commonDe from "./locales/de/common.json";
import optionsDe from "./locales/de/options.json";
import settingsDe from "./locales/de/settings.json";
import filtersDe from "./locales/de/filters.json";

export const supportedLanguages = ["en", "zh-CN", "de"] as const;
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

  if (window.navigator.language.toLowerCase().startsWith("de")) {
    return "de";
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
    de: {
      common: commonDe,
      auth: authDe,
      options: optionsDe,
      settings: settingsDe,
      filters: filtersDe
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
