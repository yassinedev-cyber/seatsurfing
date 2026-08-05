import React from "react";
import { Form, Col, Row, Button, Alert } from "react-bootstrap";
import {
  ChevronLeft as IconBack,
  Save as IconSave,
  Trash2 as IconDelete,
} from "react-feather";
import { NextRouter } from "next/router";
import FullLayout from "@/components/FullLayout";
import Loading from "@/components/Loading";
import Link from "next/link";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import Organization from "@/types/Organization";
import Domain from "@/types/Domain";
import Ajax from "@/util/Ajax";
import Client from "@/types/Client";
import RuntimeConfig from "@/components/RuntimeConfig";

import Validation from "@/util/Validation";

interface State {
  loading: boolean;
  submitting: boolean;
  saved: boolean;
  error: boolean;
  goBack: boolean;
  name: string;
  firstname: string;
  lastname: string;
  email: string;
  language: string;
  domain: string;
  password: string;
  clientSearch: string;
  clientId: string;
  attaching: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class EditOrganization extends React.Component<Props, State> {
  entity: Organization = new Organization();
  clients: Client[] = [];
  owner: Client | null = null;

  constructor(props: any) {
    super(props);
    this.state = {
      loading: true,
      submitting: false,
      saved: false,
      error: false,
      goBack: false,
      name: "",
      firstname: "",
      lastname: "",
      email: "",
      language: "en",
      domain: "",
      password: "",
      clientSearch: "",
      clientId: "",
      attaching: false,
    };
  }

  componentDidMount = () => {
    this.loadData();
    this.loadClients();
  };

  // The organization can also be handed to a client from here, which is the
  // natural direction when the organization existed first.
  loadClients = () => {
    const { id } = this.props.router.query;
    Client.list()
      .then((clients) => {
        this.clients = clients;
        this.owner =
          clients.find((c) =>
            c.organizations.some((o) => o.organizationId === id),
          ) ?? null;
        this.setState({ clientId: "" });
      })
      .catch(() => {
        this.clients = [];
      });
  };

  onAttachClient = (e: any) => {
    e.preventDefault();
    const client = this.clients.find((c) => c.id === this.state.clientId);
    if (!client) {
      return;
    }
    this.setState({ attaching: true, error: false, saved: false });
    client
      .attachOrganization(this.entity.id)
      .then(() => {
        this.setState({ attaching: false, saved: true, clientSearch: "" });
        this.loadClients();
      })
      .catch(() => this.setState({ attaching: false, error: true }));
  };

  onDetachClient = () => {
    if (!this.owner || !window.confirm(this.props.t("confirmDetachOrg"))) {
      return;
    }
    this.setState({ attaching: true, error: false, saved: false });
    this.owner
      .detachOrganization(this.entity.id)
      .then(() => {
        this.setState({ attaching: false, saved: true });
        this.loadClients();
      })
      .catch(() => this.setState({ attaching: false, error: true }));
  };

  loadData = () => {
    const { id } = this.props.router.query;
    if (id && typeof id === "string" && id !== "add") {
      Organization.get(id).then((org) => {
        this.entity = org;
        this.setState({
          name: org.name,
          firstname: org.contactFirstname,
          lastname: org.contactLastname,
          email: org.contactEmail,
          language: org.language,
          loading: false,
        });
      });
    } else {
      this.setState({ loading: false });
    }
  };

  onSubmit = (e: any) => {
    e.preventDefault();
    this.setState({
      error: false,
      saved: false,
    });
    this.entity.name = this.state.name;
    this.entity.language = this.state.language;
    const isNew = !this.entity.id;
    // A new organization is always handed to a client straight away: the admin
    // account is the client's own identity, copied by the server on attach.
    // Nothing here creates a user inside the new organization - the operator
    // has no such power, by design.
    const owner = isNew
      ? this.clients.find((c) => c.id === this.state.clientId)
      : null;
    if (isNew && !owner) {
      this.setState({ error: true });
      return;
    }
    if (isNew && owner) {
      // The organization's primary contact is the client who runs it, so it is
      // taken from the client record rather than typed in again here.
      this.entity.contactFirstname = owner.firstname;
      this.entity.contactLastname = owner.lastname;
      this.entity.contactEmail = owner.email;
    } else {
      this.entity.contactFirstname = this.state.firstname;
      this.entity.contactLastname = this.state.lastname;
      this.entity.contactEmail = this.state.email;
    }
    this.entity
      .save()
      .then(async () => {
        if (isNew && owner) {
          await owner.attachOrganization(this.entity.id);
        }
        this.props.router.push("/admin/organizations/" + this.entity.id);
        this.setState({ saved: true });
      })
      .catch(() => {
        this.setState({ error: true });
      });
  };

  deleteItem = () => {
    if (window.confirm(this.props.t("confirmDeleteOrg"))) {
      this.entity.delete().then(() => {
        this.setState({ goBack: true });
      });
    }
  };

  // Search by name or email, so a long client list stays usable.
  renderClientSection = () => {
    // The operator's own workspace is where client records live - it is never
    // handed to a client, and the server rejects the attempt anyway.
    if (!this.entity.id || this.entity.id === RuntimeConfig.INFOS.orgId) {
      return <></>;
    }
    const term = this.state.clientSearch.trim().toLowerCase();
    const matches = this.clients.filter(
      (c) =>
        term === "" ||
        c.getDisplayName().toLowerCase().includes(term) ||
        c.email.toLowerCase().includes(term),
    );
    return (
      <>
        <Form.Group as={Row} className="mt-4">
          <Form.Label column sm="6" className="lead text-uppercase">
            {this.props.t("client")}
          </Form.Label>
        </Form.Group>
        {this.owner ? (
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("belongsTo")}
            </Form.Label>
            <Col sm="4">
              <Link href={"/admin/clients/" + this.owner.id}>
                {this.owner.getDisplayName()} ({this.owner.email})
              </Link>
            </Col>
            <Col sm="2">
              <Button
                className="btn-sm"
                variant="outline-secondary"
                onClick={this.onDetachClient}
                disabled={this.state.attaching}
              >
                {this.props.t("detach")}
              </Button>
            </Col>
          </Form.Group>
        ) : (
          <Form onSubmit={this.onAttachClient} id="form-attach-client">
            <Form.Group as={Row}>
              <Form.Label column sm="2">
                {this.props.t("searchClient")}
              </Form.Label>
              <Col sm="4">
                <Form.Control
                  type="search"
                  id="client-search"
                  placeholder={this.props.t("searchClientPlaceholder")}
                  value={this.state.clientSearch}
                  onChange={(e: any) =>
                    this.setState({ clientSearch: e.target.value })
                  }
                />
              </Col>
            </Form.Group>
            <Form.Group as={Row}>
              <Form.Label column sm="2">
                {this.props.t("client")}
              </Form.Label>
              <Col sm="4">
                <Form.Select
                  id="client-select"
                  value={this.state.clientId}
                  onChange={(e: any) =>
                    this.setState({ clientId: e.target.value })
                  }
                >
                  <option value="">{this.props.t("pleaseSelect")}</option>
                  {matches.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.getDisplayName()} ({c.email})
                    </option>
                  ))}
                </Form.Select>
              </Col>
              <Col sm="2">
                <Button
                  variant="outline-secondary"
                  type="submit"
                  disabled={this.state.attaching || this.state.clientId === ""}
                >
                  {this.props.t("attach")}
                </Button>
              </Col>
            </Form.Group>
          </Form>
        )}
      </>
    );
  };

  render() {
    if (this.state.goBack) {
      this.props.router.push("/admin/organizations");
      return <></>;
    }

    const backButton = (
      <Link
        href="/admin/organizations"
        className="btn btn-sm btn-outline-secondary"
      >
        <IconBack className="feather" /> {this.props.t("back")}
      </Link>
    );
    let buttons = backButton;

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("editOrg")} buttons={buttons}>
          <Loading />
        </FullLayout>
      );
    }

    let hint = <></>;
    if (this.state.saved) {
      hint = <Alert variant="success">{this.props.t("entryUpdated")}</Alert>;
    } else if (this.state.error) {
      hint = <Alert variant="danger">{this.props.t("errorSave")}</Alert>;
    }

    const buttonDelete = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        onClick={this.deleteItem}
        disabled={false}
      >
        <IconDelete className="feather" /> {this.props.t("delete")}
      </Button>
    );
    const buttonSave = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        type="submit"
        form="form"
      >
        <IconSave className="feather" /> {this.props.t("save")}
      </Button>
    );
    if (this.entity.id) {
      buttons = (
        <>
          {backButton} {buttonDelete} {buttonSave}
        </>
      );
    } else {
      buttons = (
        <>
          {backButton} {buttonSave}
        </>
      );
    }

    const languages = ["de", "en"];

    let adminSection = <></>;
    if (!this.entity.id) {
      adminSection = (
        <>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("client")}
            </Form.Label>
            <Col sm="4">
              <Form.Select
                value={this.state.clientId}
                onChange={(e: any) =>
                  this.setState({ clientId: e.target.value })
                }
                required={true}
              >
                <option value="">{this.props.t("pleaseSelect")}</option>
                {this.clients.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.getDisplayName()} ({c.email})
                  </option>
                ))}
              </Form.Select>
              <Form.Text muted={true}>{this.props.t("orgOwnerHint")}</Form.Text>
            </Col>
          </Form.Group>
        </>
      );
    }

    return (
      <FullLayout headline={this.props.t("editOrg")} buttons={buttons}>
        <Form onSubmit={this.onSubmit} id="form">
          {hint}
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("org")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="text"
                value={this.state.name}
                onChange={(e: any) => this.setState({ name: e.target.value })}
                required={true}
                autoFocus={true}
                minLength={2}
                maxLength={64}
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("language")}
            </Form.Label>
            <Col sm="4">
              <Form.Select
                value={this.state.language}
                onChange={(e: any) =>
                  this.setState({ language: e.target.value })
                }
                required={true}
              >
                {languages.map((lc) => (
                  <option key={lc} value={lc}>
                    {this.props.t("language-" + lc)}
                  </option>
                ))}
              </Form.Select>
            </Col>
          </Form.Group>
          {/* An existing organization keeps an editable contact: it may predate
              the client model, and the address receives the deletion
              confirmation. A new one takes its contact from the client. */}
          {this.entity.id ? (
        <>
          <Form.Group as={Row}>
            <Form.Label column sm="6" className="lead text-uppercase">
              {this.props.t("primaryContact")}
            </Form.Label>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("firstname")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="text"
                value={this.state.firstname}
                onChange={(e: any) =>
                  this.setState({ firstname: e.target.value })
                }
                required={true}
                minLength={2}
                maxLength={64}
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("lastname")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="text"
                value={this.state.lastname}
                onChange={(e: any) =>
                  this.setState({ lastname: e.target.value })
                }
                required={true}
                minLength={2}
                maxLength={64}
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("emailAddress")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="email"
                value={this.state.email}
                onChange={(e: any) => this.setState({ email: e.target.value })}
                required={true}
                maxLength={128}
              />
            </Col>
          </Form.Group>
        </>
          ) : (
            <></>
          )}
          {adminSection}
        </Form>
        {this.renderClientSection()}
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(EditOrganization as any));
