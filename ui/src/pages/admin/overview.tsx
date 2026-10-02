import React from "react";
import { Table } from "react-bootstrap";
import FullLayout from "@/components/FullLayout";
import Loading from "@/components/Loading";
import Link from "next/link";
import { NextRouter } from "next/router";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import Client from "@/types/Client";
import Organization from "@/types/Organization";
import RuntimeConfig from "@/components/RuntimeConfig";

interface State {
  loading: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

interface ClientRow {
  client: Client;
  organizations: number;
  users: number;
  bookings: number;
}

/**
 * The platform operator's overview. It reports on clients - how many
 * organizations each one runs and how much they are used - and deliberately
 * never drills into the individual people inside a client's organization:
 * those belong to the client's own admin panel, not to the operator.
 */
class ClientsOverview extends React.Component<Props, State> {
  rows: ClientRow[] = [];
  totalOrganizations = 0;
  totalUsers = 0;
  totalBookings = 0;
  unassigned = 0;

  constructor(props: any) {
    super(props);
    this.state = { loading: true };
  }

  componentDidMount = () => {
    this.loadData();
  };

  loadData = () => {
    Promise.all([Client.list(), Organization.list()])
      .then(([clients, organizations]) => {
        const byId = new Map<string, Organization>();
        organizations.forEach((org) => byId.set(org.id, org));

        const owned = new Set<string>();
        this.rows = clients.map((client) => {
          let users = 0;
          let bookings = 0;
          client.organizations.forEach((co) => {
            owned.add(co.organizationId);
            const org = byId.get(co.organizationId);
            if (org) {
              users += org.userCount;
              bookings += org.bookingCount;
            }
          });
          return {
            client,
            organizations: client.organizations.length,
            users,
            bookings,
          };
        });

        this.totalOrganizations = this.rows.reduce(
          (n, r) => n + r.organizations,
          0,
        );
        this.totalUsers = this.rows.reduce((n, r) => n + r.users, 0);
        this.totalBookings = this.rows.reduce((n, r) => n + r.bookings, 0);
        // Organizations that exist but belong to no client yet. The operator's
        // own workspace is not a client organization, so it is excluded.
        this.unassigned = organizations.filter(
          (org) => !owned.has(org.id) && org.id !== RuntimeConfig.INFOS.organizationId,
        ).length;

        this.setState({ loading: false });
      })
      .catch(() => this.setState({ loading: false }));
  };

  renderTile = (label: string, value: number | string) => (
    <div className="col-6 col-md-3 mb-3" key={label}>
      <div className="card h-100">
        <div className="card-body text-center">
          <h2 className="card-title mb-0">{value}</h2>
          <p className="card-text text-muted mb-0">{label}</p>
        </div>
      </div>
    </div>
  );

  render() {
    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("operatorOverview")}>
          <Loading />
        </FullLayout>
      );
    }

    const rows = this.rows.map((row) => (
      <tr key={row.client.id}>
        <td>
          <Link href={"/admin/clients/" + row.client.id}>
            {row.client.getDisplayName()}
          </Link>
        </td>
        <td>{row.client.email}</td>
        <td>{row.organizations}</td>
        <td>{row.users}</td>
        <td>{row.bookings}</td>
      </tr>
    ));

    return (
      <FullLayout headline={this.props.t("operatorOverview")}>
        <>
          <div className="row">
            {this.renderTile(this.props.t("totalClients"), this.rows.length)}
            {this.renderTile(
              this.props.t("totalOrganizations"),
              this.totalOrganizations,
            )}
            {this.renderTile(this.props.t("totalUsers"), this.totalUsers)}
            {this.renderTile(this.props.t("totalBookings"), this.totalBookings)}
          </div>
          <h2 className="h4 mt-4">{this.props.t("perClient")}</h2>
          {rows.length === 0 ? (
            <p>{this.props.t("noRecords")}</p>
          ) : (
            <Table striped={true} hover={true}>
              <thead>
                <tr>
                  <th>{this.props.t("client")}</th>
                  <th>{this.props.t("emailAddress")}</th>
                  <th>{this.props.t("organizations")}</th>
                  <th>{this.props.t("users")}</th>
                  <th>{this.props.t("bookings")}</th>
                </tr>
              </thead>
              <tbody>{rows}</tbody>
            </Table>
          )}
          {this.unassigned > 0 ? (
            <p className="text-muted mt-3">
              <Link href="/admin/organizations">
                {this.props.t("unassignedOrgs")}: {this.unassigned}
              </Link>
            </p>
          ) : (
            <></>
          )}
        </>
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(ClientsOverview as any));
