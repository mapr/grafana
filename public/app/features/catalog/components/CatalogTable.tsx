import React from 'react';

interface TableProps {
  hover?: boolean;
  children?: React.ReactNode;
  inner?: boolean;
}

export const Table = (props: TableProps) => {
  const { hover, inner } = props;
  const classes = `service-table ${hover ? 'hover' : ''} ${inner ? 'inner' : ''}`;

  return <div className={classes}>{props.children}</div>;
};

interface RowProps {
  className?: string;
  children?: React.ReactNode;
  onClick?: () => void;
  header?: boolean;
}

export const Row = (props: RowProps) => {
  const { header, onClick, className } = props;
  const classes = `service-table-row ${className} ${header ? 'header' : ''} ${onClick ? 'clickable' : ''}`;

  return (
    <div className={classes} onClick={onClick}>
      {props.children}
    </div>
  );
};

interface ColumnProps {
  children?: React.ReactNode;
  span?: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8;
  shrink?: boolean;
}

export const Column = (props: ColumnProps) => {
  const { span, shrink } = props;
  const classes = `service-table-column ${shrink ? 'shrink' : ''} ${span ? `span-${span}` : ''}`;
  return <div className={classes}>{props.children}</div>;
};
