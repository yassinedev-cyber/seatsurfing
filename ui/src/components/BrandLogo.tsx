import React from "react";
import RuntimeConfig from "./RuntimeConfig";

/**
 * Theme-aware brand logo: renders the dark-text wordmark in light mode and
 * the white-text wordmark in dark mode (toggled via CSS). Organizations with
 * a custom logo keep a single image with the legacy dark-mode filter.
 */
export default function BrandLogo({ className }: { className?: string }) {
  const suffix = className ? " " + className : "";
  const custom = RuntimeConfig.INFOS?.customLogoUrl;
  if (custom) {
    return (
      <img src={custom} alt="Workspace" className={"brand-custom" + suffix} />
    );
  }
  return (
    <>
      <img
        src="/ui/seatsurfing.svg"
        alt="Workspace"
        className={"brand-light" + suffix}
      />
      <img
        src="/ui/seatsurfing_white.svg"
        alt="Workspace"
        className={"brand-dark" + suffix}
      />
    </>
  );
}
