import { Globe } from "../icons";
import { useTranslation } from "react-i18next";

const LANGS = [
  { code: "vi", labelKey: "language.vi" },
  { code: "en", labelKey: "language.en" },
] as const;

export function LanguageSwitcher() {
  const { t, i18n } = useTranslation();
  return (
    <label className="lang-switcher" title={t("language.label")}>
      <Globe size={14} />
      <select
        value={i18n.language?.startsWith("en") ? "en" : "vi"}
        onChange={(event) => {
          const next = event.target.value as "vi" | "en";
          i18n.changeLanguage(next);
        }}
      >
        {LANGS.map((lang) => (
          <option key={lang.code} value={lang.code}>
            {t(lang.labelKey)}
          </option>
        ))}
      </select>
    </label>
  );
}
