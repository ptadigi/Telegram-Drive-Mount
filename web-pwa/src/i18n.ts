import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import vi from "./locales/vi.json";
import en from "./locales/en.json";

const STORAGE_KEY = "td-drive.lang";
const SUPPORTED = ["vi", "en"] as const;
type Lang = (typeof SUPPORTED)[number];

function pickInitialLang(): Lang {
  if (typeof window !== "undefined") {
    const saved = window.localStorage.getItem(STORAGE_KEY);
    if (saved === "vi" || saved === "en") return saved;
    const nav = window.navigator?.language?.toLowerCase() ?? "vi";
    if (nav.startsWith("vi")) return "vi";
    if (nav.startsWith("en")) return "en";
  }
  return "vi";
}

i18n.use(initReactI18next).init({
  resources: {
    vi: { translation: vi },
    en: { translation: en },
  },
  lng: pickInitialLang(),
  fallbackLng: "vi",
  supportedLngs: SUPPORTED as unknown as string[],
  interpolation: {
    escapeValue: false,
  },
});

i18n.on("languageChanged", (lng) => {
  if (typeof window === "undefined") return;
  if (lng === "vi" || lng === "en") {
    window.localStorage.setItem(STORAGE_KEY, lng);
    document.documentElement.setAttribute("lang", lng);
  }
});

if (typeof document !== "undefined") {
  document.documentElement.setAttribute("lang", i18n.language);
}

export default i18n;
