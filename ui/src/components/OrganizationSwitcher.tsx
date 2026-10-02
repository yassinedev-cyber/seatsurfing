import React from "react";
import { Dropdown } from "react-bootstrap";
import { Check as IconCheck, ChevronDown as IconChevron } from "react-feather";
import MyOrganization from "@/types/MyOrganization";
import RuntimeConfig from "./RuntimeConfig";
import { TranslationFunc, withTranslation } from "./withTranslation";

interface Props {
  t: TranslationFunc;
}

interface State {
  organizations: MyOrganization[];
  switching: boolean;
}

/**
 * Lets an identity that belongs to more than one organization move between
 * them. Renders nothing when there is only one, so single-organization
 * clients never see it.
 */
class OrganizationSwitcher extends React.Component<Props, State> {
  constructor(props: any) {
    super(props);
    this.state = { organizations: [], switching: false };
  }

  componentDidMount = () => {
    MyOrganization.list()
      .then((organizations) => this.setState({ organizations }))
      .catch(() => this.setState({ organizations: [] }));
  };

  switchTo = (organizationId: string) => {
    this.setState({ switching: true });
    MyOrganization.switchTo(organizationId)
      .then(() => {
        // Reload so every view re-reads settings and data for the new tenant.
        window.location.href = "/ui/admin/dashboard/";
      })
      .catch(() => this.setState({ switching: false }));
  };

  render() {
    const { organizations } = this.state;
    if (organizations.length < 2) {
      return <></>;
    }
    const current =
      organizations.find((o) => o.current)?.organizationName ??
      RuntimeConfig.INFOS.orgName;
    return (
      <Dropdown className="org-switcher">
        <Dropdown.Toggle
          variant="link"
          id="org-switcher-toggle"
          disabled={this.state.switching}
        >
          <span className="org-switcher-label">
            <span className="org-switcher-eyebrow">
              {this.props.t("organization")}
            </span>
            <span className="org-switcher-name">{current}</span>
          </span>
          <IconChevron className="feather org-switcher-chevron" />
        </Dropdown.Toggle>
        <Dropdown.Menu>
          {organizations.map((org) => (
            <Dropdown.Item
              key={org.organizationId}
              active={org.current}
              onClick={() =>
                org.current ? undefined : this.switchTo(org.organizationId)
              }
            >
              {org.organizationName}
              {org.current && <IconCheck className="feather ms-2" />}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      </Dropdown>
    );
  }
}

export default withTranslation(OrganizationSwitcher as any);
