import React from 'react';

import { CatalogPod, CatalogPodStatus, PodStatusToString } from 'app/types/catalog';

const MAX_SQUARES = 7;

interface PodStatusProps {
  pods: CatalogPod[];
}

export const PodStatus = (props: PodStatusProps) => {
  const { pods } = props;

  const color = pods.some((p) => p.status === CatalogPodStatus.Red)
    ? CatalogPodStatus.Red
    : pods.some((p) => p.status === CatalogPodStatus.Yellow)
    ? CatalogPodStatus.Yellow
    : CatalogPodStatus.Green;
  return <HostStatus tall color={PodStatusToString(color)} />;
};

interface ServiceStatusProps {
  pods: CatalogPod[];
}

export const ServiceStatus = (props: ServiceStatusProps) => {
  const pods = props.pods.sort((a, b) => {
    if (a.status === b.status) {
      return 0;
    }

    if (a.status > b.status) {
      return 1;
    }

    return -1;
  });

  let squares = pods.length > MAX_SQUARES ? MAX_SQUARES : pods.length;
  const rem = squares % pods.length;

  if (rem !== 0) {
    squares -= -1;
  }

  const countPerSquare = Math.floor(pods.length / squares);

  // Determine if we need to leave space at the end for
  const podBuckets = new Array<CatalogPod[]>(rem === 0 ? squares : squares + 1);

  for (let i = 0; i < squares; i++) {
    podBuckets[i] = [];
  }

  let bucket = 0;

  pods.forEach((pod, i) => {
    podBuckets[bucket].push(pod);
    console.log(i, bucket, countPerSquare);
    if (i % countPerSquare === 0 && !!podBuckets[bucket + 1]) {
      bucket++;
    }
  });

  return (
    <div className="host-statuses">
      {podBuckets.map((p, i) => {
        return <PodStatus pods={p} key={i}></PodStatus>;
      })}
    </div>
  );
};

export interface HostStatusProps {
  color: 'green' | 'yellow' | 'red';
  tall?: boolean;
}

export const HostStatus = (props: HostStatusProps) => {
  const { color, tall } = props;
  return <div className={`host-status ${color} ${tall ? 'tall' : ''}`}></div>;
};
