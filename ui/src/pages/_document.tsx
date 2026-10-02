import Document, {
  Html,
  Head,
  Main,
  NextScript,
  DocumentProps,
} from "next/document";
import { randomBytes } from "crypto";
import RuntimeConfig from "@/components/RuntimeConfig";

type Props = DocumentProps & {
  // add custom document props
};

class Doc extends Document<Props> {
  render() {
    const nonce = randomBytes(128).toString("base64");
    const csp = new Map<string, string[]>();
    csp.set("default-src", ["'self'"]);
    csp.set("form-action", ["'self'"]);
    csp.set("img-src", ["'self'", "data:", "https:"]);
    csp.set("style-src", ["'self'", "data:", "'unsafe-inline'"]);
    csp.set("object-src", ["data:"]);
    csp.set("base-uri", ["'none'"]);
    csp.set("script-src", [
      "'self'",
      "'nonce-" + nonce + "'",
      "'strict-dynamic'",
    ]);
    if (process.env.NODE_ENV.toLowerCase() === "development") {
      csp.set("connect-src", ["'self'", "http://localhost:8080"]);
      csp.set(
        "script-src",
        Object.assign(
          [],
          csp.get("script-src")?.concat(["'unsafe-eval'", "'unsafe-inline'"]),
        ),
      );
    }
    let cspString = "";
    csp.keys().forEach((key) => {
      cspString += `${key} ${csp.get(key)?.join(" ")}; `;
    });
    return (
      <Html lang={RuntimeConfig.getLanguage()}>
        <Head nonce={nonce}>
          <meta name="robots" content="noindex" />
          <meta httpEquiv="Content-Security-Policy" content={cspString} />
          <link
            rel="preload"
            href="/ui/fonts/Inter-Var.woff2"
            as="font"
            type="font/woff2"
            crossOrigin="anonymous"
          />
          <link
            rel="preload"
            href="/ui/fonts/PlayfairDisplay-Var.woff2"
            as="font"
            type="font/woff2"
            crossOrigin="anonymous"
          />
          {/* Sets data-bs-theme before first paint to avoid a light/dark
              flash. RoyalGlass is a dark-capable design for the whole
              application, admin console included, so this deliberately does
              not exempt /admin the way upstream's own theming does. With no
              stored preference it follows the system. */}
          <script
            nonce={nonce}
            dangerouslySetInnerHTML={{
              __html:
                '(function(){try{var m=window.matchMedia("(prefers-color-scheme: dark)");var a=function(){var s=null;try{s=localStorage.getItem("rg-theme");}catch(e){}document.documentElement.setAttribute("data-bs-theme",s==="dark"||s==="light"?s:m.matches?"dark":"light");};a();if(m.addEventListener){m.addEventListener("change",a);}}catch(e){}})();',
            }}
          />
        </Head>
        <body>
          <Main />
          <NextScript nonce={nonce} />
        </body>
      </Html>
    );
  }
}

export default Doc;
