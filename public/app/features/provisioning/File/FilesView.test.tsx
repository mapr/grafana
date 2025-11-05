import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom-v5-compat';

import { Repository, useGetRepositoryFilesQuery } from 'app/api/clients/provisioning/v0alpha1';

import { FilesView } from './FilesView';

jest.mock('@grafana/i18n', () => ({
  t: (_key: string, defaultValue: string) => defaultValue,
  Trans: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

jest.mock('@grafana/ui', () => {
  const actual = jest.requireActual('@grafana/ui');

  return {
    ...actual,
    Spinner: () => <div role="status">loading</div>,
    Stack: ({ children }: { children: ReactNode }) => <div>{children}</div>,
    FilterInput: ({
      value,
      onChange,
      placeholder,
    }: {
      value: string;
      onChange: (value: string) => void;
      placeholder: string;
    }) => <input placeholder={placeholder} value={value} onChange={(event) => onChange(event.target.value)} />,
    LinkButton: ({ href, children }: { href: string; children: ReactNode }) => <a href={href}>{children}</a>,
  };
});

jest.mock('app/api/clients/provisioning/v0alpha1', () => ({
  useGetRepositoryFilesQuery: jest.fn(),
}));

const mockUseGetRepositoryFilesQuery = jest.mocked(useGetRepositoryFilesQuery);
type RepositoryFilesQueryResult = ReturnType<typeof useGetRepositoryFilesQuery>;

const baseQueryResult = (): RepositoryFilesQueryResult =>
  ({
    currentData: undefined,
    data: { items: [] },
    endpointName: 'getRepositoryFiles',
    error: undefined,
    fulfilledTimeStamp: undefined,
    isError: false,
    isFetching: false,
    isLoading: false,
    isSuccess: false,
    originalArgs: { name: '' },
    refetch: jest.fn(),
    requestId: 'test-request',
    startedTimeStamp: 0,
    status: 'uninitialized',
    subscriptionOptions: undefined,
    unsubscribe: jest.fn(),
  }) satisfies RepositoryFilesQueryResult;

const mockRepositoryFilesQuery = (overrides: Partial<RepositoryFilesQueryResult> = {}) => {
  mockUseGetRepositoryFilesQuery.mockReturnValue({
    ...baseQueryResult(),
    ...overrides,
  });
};

const defaultRepository: Repository = {
  metadata: { name: 'test-repo' },
  spec: {
    title: 'Test repository',
    type: 'github',
    workflows: ['write'],
    sync: { enabled: true, target: 'folder' },
    github: { branch: 'main' },
  },
};

const localRepository: Repository = {
  metadata: { name: 'local-repo' },
  spec: {
    title: 'Local repository',
    type: 'local',
    workflows: [],
    sync: { enabled: true, target: 'folder' },
    local: {},
  },
};

const renderComponent = (repo: Repository = defaultRepository) => {
  return render(
    <MemoryRouter>
      <FilesView repo={repo} />
    </MemoryRouter>
  );
};

describe('FilesView', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders spinner while loading', () => {
    mockRepositoryFilesQuery({ isLoading: true, status: 'pending', data: undefined });

    renderComponent();

    expect(screen.getByRole('status')).toHaveTextContent('loading');
  });

  it('renders file rows with view and history links when data is available', () => {
    mockRepositoryFilesQuery({
      isSuccess: true,
      status: 'fulfilled',
      data: {
        items: [{ path: 'dashboards/example.json', hash: 'abc', size: '10' }],
      },
    });

    renderComponent();

    const table = screen.getByRole('table');
    const getBodyRows = () =>
      within(table)
        .getAllByRole('row')
        .filter((row) => row.closest('tbody'));

    const bodyRows = getBodyRows();
    expect(bodyRows).toHaveLength(1);

    const firstDataRow = bodyRows[0];
    const rowUtils = within(firstDataRow);

    expect(rowUtils.getByText('dashboards/example.json')).toBeInTheDocument();
    expect(rowUtils.getByText((10 / 1024).toFixed(2))).toBeInTheDocument();

    const viewLink = rowUtils.getByRole('link', { name: 'View' });
    expect(viewLink).toHaveAttribute('href', '/admin/provisioning/test-repo/file/dashboards/example.json');

    const historyLink = rowUtils.getByRole('link', { name: 'History' });
    expect(historyLink).toHaveAttribute(
      'href',
      '/admin/provisioning/test-repo/history/dashboards/example.json?repo_type=github'
    );
  });

  it('filters files using search input', async () => {
    const user = userEvent.setup();

    mockRepositoryFilesQuery({
      isSuccess: true,
      status: 'fulfilled',
      data: {
        items: [
          { path: 'dashboards/example.json', hash: 'abc', size: '10' },
          { path: 'dashboards/other.yaml', hash: 'def', size: '20' },
        ],
      },
    });

    renderComponent();

    const table = screen.getByRole('table');
    const getBodyRows = () =>
      within(table)
        .getAllByRole('row')
        .filter((row) => row.closest('tbody'));

    expect(getBodyRows()).toHaveLength(2);

    const input = screen.getByPlaceholderText('Search');
    await user.clear(input);
    await user.type(input, 'other');

    await waitFor(() => {
      const rows = getBodyRows();
      expect(rows).toHaveLength(1);
      expect(within(rows[0]).getByText('dashboards/other.yaml')).toBeInTheDocument();
    });
  });

  it('hides history link when repository type is not supported', () => {
    mockRepositoryFilesQuery({
      isSuccess: true,
      status: 'fulfilled',
      data: {
        items: [{ path: 'dashboards/example.json', hash: 'abc', size: '10' }],
      },
    });

    renderComponent(localRepository);

    const table = screen.getByRole('table');
    const dataRows = within(table)
      .getAllByRole('row')
      .filter((row) => row.closest('tbody'));

    expect(dataRows).toHaveLength(1);

    const rowUtils = within(dataRows[0]);
    expect(rowUtils.getByRole('link', { name: 'View' })).toBeInTheDocument();
    expect(rowUtils.queryByRole('link', { name: 'History' })).not.toBeInTheDocument();
  });
});
