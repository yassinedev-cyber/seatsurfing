import { describe, it, expect, beforeEach } from "vitest";
import BrowserUtil from "./BrowserUtil";

const KEY = "next-export-i18n-lang";

function setQuery(query: string) {
  window.history.replaceState({}, "", "/" + query);
}

describe("BrowserUtil.applyLanguageFromQuery", () => {
  beforeEach(() => {
    window.localStorage.clear();
    setQuery("");
  });

  // This fork ships en-GB and fr only; the other language files were removed
  // deliberately, so a language that is not one of those two is unsupported.
  it("stores a supported language", () => {
    setQuery("?lang=fr");
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBe("fr");
  });

  it("stores a supported regional language", () => {
    setQuery("?lang=en-GB");
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBe("en-GB");
  });

  it("ignores an unsupported language", () => {
    window.localStorage.setItem(KEY, "fr");
    setQuery("?lang=de");
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBe("fr");
  });

  it("ignores a language without region that is not in the list", () => {
    setQuery("?lang=en");
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBeNull();
  });

  it("ignores prototype property names", () => {
    setQuery("?lang=toString");
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBeNull();
  });

  it("does nothing without lang parameter", () => {
    BrowserUtil.applyLanguageFromQuery(false);
    expect(window.localStorage.getItem(KEY)).toBeNull();
  });
});
