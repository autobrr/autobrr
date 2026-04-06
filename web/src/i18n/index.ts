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
import authRu from "./locales/ru/auth.json";
import commonRu from "./locales/ru/common.json";
import optionsRu from "./locales/ru/options.json";
import settingsRu from "./locales/ru/settings.json";
import filtersRu from "./locales/ru/filters.json";
import authDe from "./locales/de/auth.json";
import commonDe from "./locales/de/common.json";
import optionsDe from "./locales/de/options.json";
import settingsDe from "./locales/de/settings.json";
import filtersDe from "./locales/de/filters.json";
import authFr from "./locales/fr/auth.json";
import commonFr from "./locales/fr/common.json";
import optionsFr from "./locales/fr/options.json";
import settingsFr from "./locales/fr/settings.json";
import filtersFr from "./locales/fr/filters.json";
import authEs from "./locales/es/auth.json";
import commonEs from "./locales/es/common.json";
import optionsEs from "./locales/es/options.json";
import settingsEs from "./locales/es/settings.json";
import filtersEs from "./locales/es/filters.json";

export const supportedLanguages = ["en", "de", "es", "fr", "ru", "zh-CN"] as const;
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
  
  if (lang.startsWith("de")) {
    return "de";
  }

  if (lang.startsWith("es")) {
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
    ru: {
      common: commonRu,
      auth: authRu,
      options: optionsRu,
      settings: settingsRu,
      filters: filtersRu
    },
    de: {
      common: commonDe,
      auth: authDe,
      options: optionsDe,
      settings: settingsDe,
      filters: filtersDe
    },
    fr: {
      common: commonFr,
      auth: authFr,
      options: optionsFr,
      settings: settingsFr,
      filters: filtersFr
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