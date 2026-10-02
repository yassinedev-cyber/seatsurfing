import React from "react";
import ThemeSwitch from "@/components/ThemeSwitch";
import LanguageSelector from "@/components/LanguageSelector";

const CopyrightFooter: React.FC = () => {
  return (
    <div className="copyright-footer">
      &copy;&nbsp;
      <a
        href="https://seatsurfing.io"
        target="_blank"
        rel="noopener noreferrer"
      >
        Seatsurfing
      </a>
      <div className="footer-selectors">
        <ThemeSwitch />
        <LanguageSelector compactBreakpoint="md" />
      </div>
    </div>
  );
};

export default CopyrightFooter;
