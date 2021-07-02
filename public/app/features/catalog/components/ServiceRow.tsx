import React, { useState, useMemo } from 'react';
import { ServiceComponent, CatalogPod, PodStatusToString } from 'app/types/catalog';
import { ServiceStatus, HostStatus } from '../components/ServiceStatus';
import { Icon, Tooltip } from '@grafana/ui';
import { Table, Row, Column } from './CatalogTable';
import { ServiceLabels } from './ServiceLabels';

interface Props {
  service: string;
  component: ServiceComponent;
  collapsible?: boolean;
  collapsed?: boolean;
  onClick?: () => void;
}

export const ServiceHeader = () => {
  return (
    <Row header>
      <Column span={1}></Column>
      <Column span={5}>Namespace</Column>
      <Column span={5}>Service</Column>
      <Column span={5}>Application</Column>
      <Column span={5}>Status</Column>
      <Column span={8}>Address</Column>
      <Column span={8}>Labels</Column>
    </Row>
  );
};

export const ServiceRow = (props: Props) => {
  const { collapsible, component, service } = props;

  const [open, setOpen] = useState<boolean>(false);

  const angleIcon = useMemo(() => {
    return open ? <Icon name="angle-up" /> : <Icon name="angle-down" />;
  }, [open]);

  return (
    <>
      <Row onClick={() => setOpen(!open)}>
        <Column span={1}>{collapsible && angleIcon}</Column>
        <Column span={5}>{component.namespace}</Column>
        <Column span={5}>{service}</Column>
        <Column span={5}>application</Column>
        <Tooltip show={true} theme="info-alt" content={<span>test?</span>}>
          <Column span={5}>
            <ServiceStatus pods={component.pods || []} />
          </Column>
        </Tooltip>
        <Column span={8}>{component.address}</Column>
        <Column span={8}>
          <ServiceLabels labels={component.labels} />
        </Column>
      </Row>
      {open && component.pods && <HostsRow hosts={component.pods} />}
    </>
  );
};

interface HostRowProps {
  hosts: CatalogPod[];
}

export const HostsRow = (props: HostRowProps) => {
  const { hosts } = props;
  return (
    <Table inner>
      {hosts.map((host, i) => {
        return (
          <Row key={i}>
            <Column shrink span={1}>
              <HostStatus color={PodStatusToString(host.status)} />
            </Column>
            <Column span={8}>{host.name}</Column>
          </Row>
        );
      })}
    </Table>
  );
};
