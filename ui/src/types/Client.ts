import Ajax from "../util/Ajax";

/**
 * An organization a client owns. The client holds one OrgAdmin account per
 * organization, all sharing the client's email address.
 */
export class ClientOrganization {
  organizationId: string;
  organizationName: string;
  userId: string;

  constructor() {
    this.organizationId = "";
    this.organizationName = "";
    this.userId = "";
  }

  deserialize(input: any): void {
    this.organizationId = input.organizationId;
    this.organizationName = input.organizationName;
    this.userId = input.userId;
  }
}

/**
 * A client is a customer of the platform. The record lives in the operator's
 * own organization and acts as a directory entry - the organizations the
 * client owns are attached to it, either right away or at any later point.
 */
export default class Client {
  id: string;
  email: string;
  firstname: string;
  lastname: string;
  password: string;
  organizations: ClientOrganization[];

  constructor() {
    this.id = "";
    this.email = "";
    this.firstname = "";
    this.lastname = "";
    this.password = "";
    this.organizations = [];
  }

  deserialize(input: any): void {
    this.id = input.id;
    this.email = input.email;
    this.firstname = input.firstname;
    this.lastname = input.lastname;
    this.organizations = [];
    (input.organizations ?? []).forEach((item: any) => {
      const org = new ClientOrganization();
      org.deserialize(item);
      this.organizations.push(org);
    });
  }

  getDisplayName(): string {
    const name = `${this.firstname} ${this.lastname}`.trim();
    return name !== "" ? name : this.email;
  }

  serialize(): Object {
    const obj: any = {
      email: this.email,
      firstname: this.firstname,
      lastname: this.lastname,
      // The directory entry is a plain user in the operator's organization.
      // Admin rights are granted per owned organization on attach, so a client
      // never gains any power over the operator's own workspace.
      role: 0,
    };
    if (this.password !== "") {
      obj.password = this.password;
    }
    return obj;
  }

  async save(): Promise<Client> {
    if (this.id === "") {
      const result = await Ajax.postData("/user/", this.serialize());
      this.id = result.objectId;
    } else {
      await Ajax.putData(
        `/user/${encodeURIComponent(this.id)}`,
        this.serialize(),
      );
    }
    return this;
  }

  async delete(): Promise<void> {
    await Ajax.delete(`/user/${encodeURIComponent(this.id)}`);
  }

  /** Links an organization the client already owns, or a brand new one. */
  async attachOrganization(organizationId: string): Promise<void> {
    await Ajax.postData(`/user/${encodeURIComponent(this.id)}/organizations`, {
      organizationId: organizationId,
    });
  }

  /** Removes the client's admin account from an organization. Data is kept. */
  async detachOrganization(organizationId: string): Promise<void> {
    await Ajax.delete(
      `/user/${encodeURIComponent(this.id)}/organizations/${encodeURIComponent(
        organizationId,
      )}`,
    );
  }

  static async list(): Promise<Client[]> {
    const result = await Ajax.get("/user/clients");
    const list: Client[] = [];
    ((result.json as []) ?? []).forEach((item) => {
      const e: Client = new Client();
      e.deserialize(item);
      list.push(e);
    });
    return list;
  }

  static async get(id: string): Promise<Client | null> {
    const list = await Client.list();
    return list.find((c) => c.id === id) ?? null;
  }
}
