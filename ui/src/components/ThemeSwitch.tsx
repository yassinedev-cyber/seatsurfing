import React, { useEffect, useState } from "react";
import { Sun as IconSun, Moon as IconMoon } from "react-feather";
import { useTranslation } from "next-export-i18n";

export default function ThemeSwitch() {
  const { t } = useTranslation();
  const [theme, setTheme] = useState<"light" | "dark">("light");

  useEffect(() => {
    setTheme(
      document.documentElement.getAttribute("data-bs-theme") === "dark"
        ? "dark"
        : "light",
    );
  }, []);

  const toggle = () => {
    const next = theme === "dark" ? "light" : "dark";
    setTheme(next);
    document.documentElement.setAttribute("data-bs-theme", next);
    try {
      window.localStorage.setItem("rg-theme", next);
    } catch (e) {
      // localStorage not available; theme still applies for this session
    }
  };

  return (
    <button
      type="button"
      className="rg-theme-switch"
      role="switch"
      aria-checked={theme === "dark"}
      aria-label={t("themeToggle")}
      onClick={toggle}
    >
      <IconSun className="feather rg-sun" />
      <IconMoon className="feather rg-moon" />
      <span className="rg-thumb" />
    </button>
  );
}
