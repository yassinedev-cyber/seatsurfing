import React from "react";
import { Table } from "react-bootstrap";
import { Plus as IconPlus } from "react-feather";
import FullLayout from "@/components/FullLayout";
import Loading from "@/components/Loading";
import Link from "next/link";
import { NextRouter } from "next/router";
import withReadyRouter from "@/components/withReadyRouter";
import { TranslationFunc, withTranslation } from "@/components/withTranslation";
import Client from "@/types/Client";

interface State {
  selectedItem: string;
  loading: boolean;
}

interface Props {
  router: NextRouter;
  t: TranslationFunc;
}

class Clients extends React.Component<Props, State> {
  data: Client[] = [];

  constructor(props: any) {
    super(props);
    this.state = {
      selectedItem: "",
      loading: true,
    };
  }

  componentDidMount = () => {
    this.loadItems();
  };

  loadItems = () => {
    Client.list().then((list) => {
      this.data = list;
      this.setState({ loading: false });
    });
  };

  onItemSelect = (client: Client) => {
    this.setState({ selectedItem: client.id });
  };

  renderItem = (client: Client) => {
    const orgNames = client.organizations
      .map((org) => org.organizationName)
      .join(", ");
    return (
      <tr key={client.id} onClick={() => this.onItemSelect(client)}>
        <td>{client.getDisplayName()}</td>
        <td>{client.email}</td>
        <td>{orgNames !== "" ? orgNames : this.props.t("noOrganizations")}</td>
      </tr>
    );
  };

  render() {
    if (this.state.selectedItem) {
      this.props.router.push(`/admin/clients/${this.state.selectedItem}`);
      return <></>;
    }

    let buttons = (
      <Link
        href="/admin/clients/add"
        className="btn btn-sm btn-outline-secondary"
      >
        <IconPlus className="feather" /> {this.props.t("add")}
      </Link>
    );

    if (this.state.loading) {
      return (
        <FullLayout headline={this.props.t("clients")} buttons={buttons}>
          <Loading />
        </FullLayout>
      );
    }

    let rows = this.data.map((item) => this.renderItem(item));
    if (rows.length === 0) {
      return (
        <FullLayout headline={this.props.t("clients")} buttons={buttons}>
          <p>{this.props.t("noRecords")}</p>
        </FullLayout>
      );
    }
    return (
      <FullLayout headline={this.props.t("clients")} buttons={buttons}>
        <Table striped={true} hover={true} className="clickable-table">
          <thead>
            <tr>
              <th>{this.props.t("name")}</th>
              <th>{this.props.t("emailAddress")}</th>
              <th>{this.props.t("organizations")}</th>
            </tr>
          </thead>
          <tbody>{rows}</tbody>
        </Table>
      </FullLayout>
    );
  }
}

export default withTranslation(withReadyRouter(Clients as any));
