import { css } from '@emotion/css';

import { t } from '@grafana/i18n';
import { FileDropzone, IconButton, Stack, Text } from '@grafana/ui';

export const YAML_FILE_EXTENSIONS = ['.yaml', '.yml'];

// FileDropzone has no prop to suppress its built-in "Accepted file types" caption, shown via Field description instead.
const hideAcceptedFileTypesCaption = css({
  small: {
    display: 'none',
  },
});

// FileDropzone's `accept` option buckets extensions by MIME type, so a `.txt` file (MIME
// text/plain) collides with the `.yml` bucket and passes; re-check the filename directly.
function validateYamlFileExtension(file: File) {
  const hasYamlExtension = YAML_FILE_EXTENSIONS.some((ext) => file.name.toLowerCase().endsWith(ext));
  if (hasYamlExtension) {
    return null;
  }
  return {
    code: 'file-invalid-type',
    message: t('alerting.import-to-gma.yaml-file.invalid-type', 'File must be a YAML file ({{extensions}})', {
      extensions: YAML_FILE_EXTENSIONS.join(', '),
    }),
  };
}

/** The "Accepted file types" text to show as the Field description, since the dropzone's own is hidden. */
export function getYamlFileDescription() {
  return t('alerting.import-to-gma.yaml-file.description', 'Accepted file types: {{extensions}}', {
    extensions: YAML_FILE_EXTENSIONS.join(', '),
  });
}

interface YamlFileDropzoneProps {
  file: File | null | undefined;
  onChange: (file: File | null) => void;
  /** Called after a file is accepted, for example to confirm the upload. */
  onUploaded?: (file: File) => void;
}

/**
 * Drop zone for a single YAML file, with the chosen file listed below it and a button to remove it.
 * Wrap it in a Field that uses getYamlFileDescription() as its description.
 */
export function YamlFileDropzone({ file, onChange, onUploaded }: YamlFileDropzoneProps) {
  return (
    <Stack direction="column" gap={1}>
      <div className={hideAcceptedFileTypesCaption}>
        <FileDropzone
          options={{
            multiple: false,
            accept: YAML_FILE_EXTENSIONS,
            validator: validateYamlFileExtension,
            onDrop: (acceptedFiles) => {
              const accepted = acceptedFiles[0];
              if (accepted) {
                onChange(accepted);
                onUploaded?.(accepted);
              }
            },
          }}
          fileListRenderer={() => null}
        >
          <Text color="secondary">
            {t('alerting.import-to-gma.yaml-file.upload', 'Drop YAML file here or click to upload')}
          </Text>
        </FileDropzone>
      </div>

      {file && (
        <Stack direction="row" alignItems="center" justifyContent="space-between">
          <Text>{file.name}</Text>
          <IconButton
            name="trash-alt"
            tooltip={t('alerting.import-to-gma.yaml-file.remove', 'Remove {{name}}', { name: file.name })}
            onClick={() => onChange(null)}
          />
        </Stack>
      )}
    </Stack>
  );
}
