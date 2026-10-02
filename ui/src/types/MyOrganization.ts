import Ajax from "../util/Ajax";
import AjaxCredentials from "../util/AjaxCredentials";
import JwtDecoder from "../util/JwtDecoder";

/**
 * An organization the signed-in identity belongs to. A client who owns several
 * organizations holds one user record per organization, all sharing an email,
 * and can move between them without signing in again.
 */
export default class MyOrganization {
  organizationId: string;
  organizationName: string;
  current: boolean;

  constructor() {
    this.organizationId = "";
    this.organizationName = "";
    this.current = false;
  }

  deserialize(input: any): void {
    this.organizationId = input.organizationId;
    this.organizationName = input.organizationName;
    this.current = input.current;
  }

  static async list(): Promise<MyOrganization[]> {
    const result = await Ajax.get("/user/organizations");
    if (!result.json || !Array.isArray(result.json)) {
      return [];
    }
    return result.json.map((item: any) => {
      const org = new MyOrganization();
      org.deserialize(item);
      return org;
    });
  }

  /** Exchanges the current session for one in the target organization. */
  static async switchTo(organizationId: string): Promise<void> {
    const result = await Ajax.postData(
      `/user/organizations/${encodeURIComponent(organizationId)}/switch`,
    );
    const credentials: AjaxCredentials = {
      accessToken: result.json.accessToken,
      accessTokenExpiry: JwtDecoder.getExpiryDate(result.json.accessToken),
      logoutUrl: result.json.logoutUrl ?? "",
      profilePageUrl: result.json.profilePageUrl ?? "",
    };
    Ajax.PERSISTER.updateCredentialsLocalStorage(credentials);
    Ajax.PERSISTER.persistRefreshTokenInLocalStorage(result.json.refreshToken);
  }
}
