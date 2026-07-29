const i18n = {
  translations: {
    "en-GB": require("./translations.en-GB.json"),
    fr: require("./translations.fr.json"),
  },
  defaultLang: "en-GB",
  useBrowserDefault: true,
  languageDataStore: "localStorage",
};

module.exports = i18n;
