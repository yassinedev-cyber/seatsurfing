import React from "react";
import { Badge, Button, Table } from "react-bootstrap";
import { Plus as IconPlus } from "react-feather";
import FullLayout from "@/components/FullLayout";
import Loading from "@/components/Loading";
import Link from "next/link";
import { NextRouter } from "next/router";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import Organization from "@/types/Organization";
import Client from "@/types/Client";
import MyOrganization from "@/types/MyOrganization";
import RuntimeConfig from "@/components/RuntimeConfig";

interface State {
  selectedItem: string;
  loading: boolean;
  switching: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class Organizations extends React.Component<Props, State> {
  data: Organization[] = [];
  // organization id -> the client who owns it, so the operator can see at a
  // glance which organizations still belong to nobody
  owners: Map<string, string> = new Map();

  constructor(props: any) {
    super(props);
    this.state = {
      selectedItem: "",
      loading: true,
      switching: false,
    };
  }

  isPlatformOperator = () => RuntimeConfig.INFOS.superAdmin;

  componentDidMount = () => {
    this.loadItems();
  };

  loadItems = () => {
    // The same listing answers differently for a client: their own
    // organizations, without the ownership column, which is the operator's
    // view of the customer base and not a client's business.
    if (!this.isPlatformOperator()) {
      Organization.list()
        .then((organizations) => {
          this.data = organizations;
          this.setState({ loading: false });
        })
        .catch(() => this.setState({ loading: false }));
      return;
    }
    Promise.all([Organization.list(), Client.list()])
      .then(([organizations, clients]) => {
        this.data = organizations;
        this.owners = new Map();
        clients.forEach((client) => {
          client.organizations.forEach((org) => {
            this.owners.set(org.organizationId, client.getDisplayName());
          });
        });
        this.setState({ loading: false });
      })
      .catch(() => this.setState({ loading: false }));
  };

  onItemSelect = (org: Organization) => {
    this.setState({ selectedItem: org.id });
  };

  switchTo = (organizationId: string) => {
    this.setState({ switching: true });
    MyOrganization.switchTo(organizationId)
      .then(() => {
        // A full load, not a route change: every view has to re-read its
        // settings and data for the organization now in session.
        window.location.href = "/ui/admin/organizations/";
      })
      .catch(() => this.setState({ switching: false }));
  };

  renderItem = (org: Organization) => {
    const owner = this.owners.get(org.id);
    const isOwnWorkspace = org.id === RuntimeConfig.INFOS.orgId;
    return (
      <tr key={org.id} onClick={() => this.onItemSelect(org)}>
        <td>{org.name}</td>
        <td>
          {owner ?? (
            <span className="text-muted">
              {isOwnWorkspace
                ? this.props.t("operatorWorkspace")
                : this.props.t("noClient")}
            </span>
          )}
        </td>
        <td>{org.userCount}</td>
        <td>{org.bookingCount}</td>
      </tr>
    );
  };

  // A client's own organization is the one they can walk into and administer.
  // The others are one click away, so the row offers the switch rather than an
  // edit page that would refuse them.
  renderOwnItem = (org: Organization) => {
    const isCurrent = org.id === RuntimeConfig.INFOS.orgId;
    return (
      <tr key={org.id}>
        <td>
          {isCurrent ? (
            <Link href={"/admin/organizations/" + org.id}>{org.name}</Link>
          ) : (
            org.name
          )}
        </td>
        <td>{org.userCount}</td>
        <td>{org.bookingCount}</td>
        <td>
          {isCurrent ? (
            <Badge bg="primary">{this.props.t("current")}</Badge>
          ) : (
            <Button
              className="btn-sm"
              variant="outline-secondary"
              disabled={this.state.switching}
              onClick={() => this.switchTo(org.id)}
            >
              {this.props.t("switchOrg")}
            </Button>
          )}
        </td>
      </tr>
    );
  };

  render() {
    if (this.state.selectedItem) {
      this.props.router.push(`/admin/organizations/${this.state.selectedItem}`);
      return <></>;
    }

    let buttons = (
      <Link
        href="/admin/organizations/add"
        className="btn btn-sm btn-outline-secondary"
      >
        <IconPlus className="feather" /> {this.props.t("add")}
      </Link>
    );

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("organizations")} buttons={buttons}>
          <Loading />
        </FullLayout>
      );
    }

    const isOperator = this.isPlatformOperator();
    let rows = this.data.map((item) =>
      isOperator ? this.renderItem(item) : this.renderOwnItem(item),
    );
    if (rows.length === 0) {
      return (
        <FullLayout headline={this.props.t("organizations")} buttons={buttons}>
          <p>{this.props.t("noRecords")}</p>
        </FullLayout>
      );
    }
    return (
      <FullLayout headline={this.props.t("organizations")} buttons={buttons}>
        {!isOperator && <p className="text-muted">{this.props.t("orgIsolationHint")}</p>}
        <Table
          striped={true}
          hover={true}
          className={isOperator ? "clickable-table" : ""}
        >
          <thead>
            <tr>
              <th>{this.props.t("org")}</th>
              {isOperator && <th>{this.props.t("client")}</th>}
              <th>{this.props.t("users")}</th>
              <th>{this.props.t("bookings")}</th>
              {!isOperator && <th></th>}
            </tr>
          </thead>
          <tbody>{rows}</tbody>
        </Table>
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(Organizations as any));
