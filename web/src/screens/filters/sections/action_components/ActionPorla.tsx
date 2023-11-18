import * as Input from "@components/inputs";

import { CollapsibleSection } from "../_components";
import * as FilterSection from "../_components";

export const Porla = ({ idx, action, clients }: ClientActionProps) => (
  <>
    <FilterSection.Section
      title="Instance"
      subtitle={
        <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
      }
    >
      <FilterSection.Layout>
        <FilterSection.HalfRow>
          <Input.DownloadClientSelect
            name={`actions.${idx}.client_id`}
            action={action}
            clients={clients}
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
          <Input.TextField
            name={`actions.${idx}.label`}
            label="Preset"
            placeholder="eg. default"
            tooltip={
              <div>A case-sensitive preset name as configured in Porla.</div>
            }
          />
        </FilterSection.HalfRow>
      </FilterSection.Layout>

      <Input.TextAreaAutoResize
        name={`actions.${idx}.save_path`}
        label="Save path"
        placeholder="eg. /full/path/to/torrent/data"
        className="pb-6"
      />

      <CollapsibleSection
        noBottomBorder
        title="Limits"
        subtitle="Configure your speed/ratio/seed time limits"
      >
        <FilterSection.HalfRow>
          <Input.NumberField
            name={`actions.${idx}.limit_download_speed`}
            label="Limit download speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterSection.HalfRow>
        <FilterSection.HalfRow>
          <Input.NumberField
            name={`actions.${idx}.limit_upload_speed`}
            label="Limit upload speed (KiB/s)"
            placeholder="Takes any number (0 is no limit)"
          />
        </FilterSection.HalfRow>
      </CollapsibleSection>
    </FilterSection.Section>
  </>
);
