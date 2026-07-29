import React from "react";
import { TranslationFunc, withTranslation } from "./withTranslation";
import RuntimeConfig from "./RuntimeConfig";
import { LanguageSwitcher } from "next-export-i18n";

interface State {}

interface Props {
  inNavbar?: boolean;
  drop?: "up" | "down" | "start" | "end";
  align?: "start" | "end";
  t: TranslationFunc;
}

class LanguageSelector extends React.Component<Props, State> {
  render() {
    const current = RuntimeConfig.getLanguage();
    const languages = Object.entries(RuntimeConfig.getAvailableLanguages());

    return (
      <div className="rg-lang-switch" role="group" aria-label="Language">
        {languages.map(([code, name]) => (
          <LanguageSwitcher key={"lng-" + code} lang={code}>
            <button
              type="button"
              className={"rg-lang-pill" + (code === current ? " active" : "")}
              aria-pressed={code === current}
              title={name}
            >
              {code.slice(0, 2).toUpperCase()}
            </button>
          </LanguageSwitcher>
        ))}
      </div>
    );
  }
}

export default withTranslation(LanguageSelector as any);
