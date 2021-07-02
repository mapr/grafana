export type Catalog = CatalogService[];

export interface CatalogService {
  name: string;
  components: ServiceComponent[];
}

export type CatalogLabels = {
  [key: string]: string;
};

export enum CatalogPodStatus {
  Green = 0,
  Yellow = 1,
  Red = 2,
}

export const PodStatusToString = (status: CatalogPodStatus): 'red' | 'green' | 'yellow' => {
  switch (status) {
    case CatalogPodStatus.Green:
      return 'green';
    case CatalogPodStatus.Red:
      return 'red';
    case CatalogPodStatus.Yellow:
      return 'yellow';
  }
};

export interface CatalogPod {
  name: string;
  status: CatalogPodStatus;
}

export interface ServiceComponent {
  name: string;
  labels?: CatalogLabels;
  teams: string[];
  namespace: string;
  address: string;
  pods?: CatalogPod[];
  external: boolean;
}

export interface CatalogState {
  catalog: Catalog;
}
