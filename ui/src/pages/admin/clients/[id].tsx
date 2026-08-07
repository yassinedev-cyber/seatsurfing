import React from "react";
import { Form, Col, Row, Button, Alert, Table } from "react-bootstrap";
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
import Client from "@/types/Client";
import Organization from "@/types/Organization";
import Domain from "@/types/Domain";
import Validation from "@/util/Validation";

interface State {
  loading: boolean;
  submitting: boolean;
  saved: boolean;
  error: boolean;
  goBack: boolean;
  firstname: string;
  lastname: string;
  email: string;
  password: string;
  // inline organization, created together with the client or added later
  orgName: string;
  orgDomain: string;
  attachOrgId: string;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class EditClient extends React.Component<Props, State> {
  entity: Client = new Client();
  unattachedOrgs: Organization[] = [];

  constructor(props: any) {
    super(props);
    this.state = {
      loading: true,
      submitting: false,
      saved: false,
      error: false,
      goBack: false,
      firstname: "",
      lastname: "",
      email: "",
      password: "",
      orgName: "",
      orgDomain: "",
      attachOrgId: "",
    };
  }

  componentDidMount = () => {
    this.loadData();
  };

  loadData = () => {
    const { id } = this.props.router.query;
    if (id && typeof id === "string" && id !== "add") {
      Client.get(id).then((client) => {
        if (!client) {
          this.setState({ loading: false, error: true });
          return;
        }
        this.entity = client;
        this.setState({
          firstname: client.firstname,
          lastname: client.lastname,
          email: client.email,
        });
        this.loadUnattachedOrgs();
      });
    } else {
      this.setState({ loading: false });
    }
  };

  // Organizations that exist but are not owned by this client yet - these are
  // the candidates for attaching an organization after the fact.
  loadUnattachedOrgs = () => {
    Organization.list().then((list) => {
      const owned = this.entity.organizations.map((o) => o.organizationId);
      this.unattachedOrgs = list.filter((org) => owned.indexOf(org.id) < 0);
      this.setState({ loading: false });
    });
  };

  onSubmit = (e: any) => {
    e.preventDefault();
    this.setState({ error: false, saved: false, submitting: true });
    this.entity.firstname = this.state.firstname;
    this.entity.lastname = this.state.lastname;
    this.entity.email = this.state.email;
    this.entity.password = this.state.password;
    const isNew = this.entity.id === "";
    this.entity
      .save()
      .then(async () => {
        // A client is an administrator, so they always get a workspace to
        // administer: without one they would sign in to nothing.
        if (isNew) {
          await this.createAndAttachOrg();
        }
        this.setState({
          submitting: false,
          saved: true,
          password: "",
          orgName: "",
          orgDomain: "",
        });
        if (isNew) {
          // Same route, so the component is re-used rather than re-mounted:
          // reload explicitly or the page would keep showing the create form.
          this.props.router.push("/admin/clients/" + this.entity.id);
        }
        this.reload();
      })
      .catch(() => {
        this.setState({ error: true, submitting: false });
      });
  };

  // Creates a fresh organization for this client and hands them an admin
  // account in it. Used both for the organization created alongside the client
  // and for organizations added later.
  createAndAttachOrg = async () => {
    const org = new Organization();
    // Naming the workspace is optional: a client who runs a single workspace
    // has no reason to think about it, so it takes their own name by default.
    org.name = this.state.orgName.trim() || this.entity.getDisplayName();
    org.contactFirstname = this.entity.firstname;
    org.contactLastname = this.entity.lastname;
    org.contactEmail = this.entity.email;
    // Language is chosen per person in the interface, not per workspace. The
    // server still requires a value, so every workspace gets the same one.
    org.language = Organization.DEFAULT_LANGUAGE;
    await org.save();
    // No domain is registered: organizations share the platform's single
    // sign-in address and are resolved from the email address at login.
    await this.entity.attachOrganization(org.id);
  };

  onAddOrganization = (e: any) => {
    e.preventDefault();
    this.setState({ error: false, saved: false, submitting: true });
    this.createAndAttachOrg()
      .then(() => {
        this.setState({
          submitting: false,
          saved: true,
          orgName: "",
          orgDomain: "",
        });
        this.reload();
      })
      .catch(() => {
        this.setState({ error: true, submitting: false });
      });
  };

  onAttachExisting = (e: any) => {
    e.preventDefault();
    if (this.state.attachOrgId === "") {
      return;
    }
    this.setState({ error: false, saved: false, submitting: true });
    this.entity
      .attachOrganization(this.state.attachOrgId)
      .then(() => {
        this.setState({ submitting: false, saved: true, attachOrgId: "" });
        this.reload();
      })
      .catch(() => {
        this.setState({ error: true, submitting: false });
      });
  };

  onDetach = (organizationId: string) => {
    if (!window.confirm(this.props.t("confirmDetachOrg"))) {
      return;
    }
    this.entity
      .detachOrganization(organizationId)
      .then(() => {
        this.setState({ saved: true });
        this.reload();
      })
      .catch(() => {
        this.setState({ error: true });
      });
  };

  reload = () => {
    Client.get(this.entity.id).then((client) => {
      if (client) {
        this.entity = client;
      }
      this.loadUnattachedOrgs();
    });
  };

  deleteItem = () => {
    if (window.confirm(this.props.t("confirmDeleteClient"))) {
      this.entity.delete().then(() => {
        this.setState({ goBack: true });
      });
    }
  };

  renderOrgRow = (org: {
    organizationId: string;
    organizationName: string;
  }) => {
    return (
      <tr key={org.organizationId}>
        <td>
          <Link href={"/admin/organizations/" + org.organizationId}>
            {org.organizationName}
          </Link>
        </td>
        <td>
          <Button
            className="btn-sm"
            variant="outline-secondary"
            onClick={() => this.onDetach(org.organizationId)}
          >
            {this.props.t("detach")}
          </Button>
        </td>
      </tr>
    );
  };

  render() {
    if (this.state.goBack) {
      this.props.router.push("/admin/clients");
      return <></>;
    }

    const backButton = (
      <Link href="/admin/clients" className="btn btn-sm btn-outline-secondary">
        <IconBack className="feather" /> {this.props.t("back")}
      </Link>
    );
    let buttons = backButton;

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("editClient")} buttons={buttons}>
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

    const buttonSave = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        type="submit"
        form="form"
        disabled={this.state.submitting}
      >
        <IconSave className="feather" /> {this.props.t("save")}
      </Button>
    );
    const buttonDelete = (
      <Button
        className="btn-sm"
        variant="outline-secondary"
        onClick={this.deleteItem}
      >
        <IconDelete className="feather" /> {this.props.t("delete")}
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
    const isNew = this.entity.id === "";

    // While creating the client the organization is optional: it can be filled
    // in here, or added later from this same page once the client exists.
    let inlineOrgSection = <></>;
    if (isNew) {
      inlineOrgSection = (
        <>
          <Form.Group as={Row}>
            <Form.Label column sm="6" className="lead text-uppercase">
              {this.props.t("organization")}
            </Form.Label>
          </Form.Group>
          {this.renderOrgFields()}
        </>
      );
    }

    let orgSection = <></>;
    if (!isNew) {
      const rows = this.entity.organizations.map((org) =>
        this.renderOrgRow(org),
      );
      orgSection = (
        <>
          <Form.Group as={Row} className="mt-4">
            <Form.Label column sm="6" className="lead text-uppercase">
              {this.props.t("organizations")}
            </Form.Label>
          </Form.Group>
          {rows.length > 0 ? (
            <Table striped={true} hover={true}>
              <thead>
                <tr>
                  <th>{this.props.t("org")}</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>{rows}</tbody>
            </Table>
          ) : (
            <p>{this.props.t("noOrganizations")}</p>
          )}

          <Form onSubmit={this.onAttachExisting} id="form-attach">
            <Form.Group as={Row}>
              <Form.Label column sm="2">
                {this.props.t("attachOrg")}
              </Form.Label>
              <Col sm="4">
                <Form.Select
                  value={this.state.attachOrgId}
                  onChange={(e: any) =>
                    this.setState({ attachOrgId: e.target.value })
                  }
                >
                  <option value="">{this.props.t("pleaseSelect")}</option>
                  {this.unattachedOrgs.map((org) => (
                    <option key={org.id} value={org.id}>
                      {org.name}
                    </option>
                  ))}
                </Form.Select>
              </Col>
              <Col sm="2">
                <Button
                  variant="outline-secondary"
                  type="submit"
                  disabled={
                    this.state.submitting || this.state.attachOrgId === ""
                  }
                >
                  {this.props.t("attach")}
                </Button>
              </Col>
            </Form.Group>
          </Form>

          <Form onSubmit={this.onAddOrganization} id="form-add-org">
            <Form.Group as={Row} className="mt-4">
              <Form.Label column sm="6" className="lead text-uppercase">
                {this.props.t("addOrg")}
              </Form.Label>
            </Form.Group>
            {this.renderOrgFields(true)}
            <Form.Group as={Row}>
              <Col sm={{ span: 4, offset: 2 }}>
                <Button
                  variant="outline-secondary"
                  type="submit"
                  disabled={this.state.submitting || this.state.orgName === ""}
                >
                  {this.props.t("addOrg")}
                </Button>
              </Col>
            </Form.Group>
          </Form>
        </>
      );
    }

    return (
      <FullLayout
        headline={
          isNew ? this.props.t("addClient") : this.props.t("editClient")
        }
        buttons={buttons}
      >
        <Form onSubmit={this.onSubmit} id="form">
          {hint}
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
                autoFocus={true}
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
              />
            </Col>
          </Form.Group>
          <Form.Group as={Row}>
            <Form.Label column sm="2">
              {this.props.t("password")}
            </Form.Label>
            <Col sm="4">
              <Form.Control
                type="password"
                value={this.state.password}
                onChange={(e: any) =>
                  this.setState({ password: e.target.value })
                }
                required={isNew}
                minLength={Validation.PASSWORD_MIN_LENGTH}
                maxLength={Validation.PASSWORD_MAX_LENGTH}
                pattern={Validation.PASSWORD_PATTERN}
                title={this.props.t("passwordRequirements")}
              />
              <Form.Text muted={true}>
                {this.props.t("clientCredentialsHint")}
                <br />
                {this.props.t("passwordRequirements")}
              </Form.Text>
            </Col>
          </Form.Group>
          {inlineOrgSection}
        </Form>
        {orgSection}
      </FullLayout>
    );
  }

  renderOrgFields = (required: boolean = false) => {
    return (
      <Form.Group as={Row}>
        <Form.Label column sm="2">
          {this.props.t("org")}
        </Form.Label>
        <Col sm="4">
          <Form.Control
            type="text"
            value={this.state.orgName}
            onChange={(e: any) => this.setState({ orgName: e.target.value })}
            required={required}
            maxLength={64}
            placeholder={required ? "" : this.props.t("orgNamePlaceholder")}
          />
          {!required && (
            <Form.Text muted={true}>{this.props.t("orgNameHint")}</Form.Text>
          )}
        </Col>
      </Form.Group>
    );
  };
}

export default withTranslation(withReadyRouter(EditClient as any));
