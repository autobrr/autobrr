import { useQuery } from "@tanstack/react-query";

import { APIClient } from "@api/APIClient";
import { downloadsPerUnitOptions } from "@domain/constants";

import { DocsLink } from "@components/ExternalLink";

import * as Input from "@components/inputs";
import * as Components from "./_components";

const MapIndexer = (indexer: Indexer) => (
  { label: indexer.name, value: indexer.id } as Input.MultiSelectOption
);

export const General = () => {
  const { isLoading, data } = useQuery({
    queryKey: ["filters", "indexer_list"],
    queryFn: APIClient.indexers.getOptions,
    refetchOnWindowFocus: false
  });

  const indexerOptions = data?.map(MapIndexer) ?? [];

  return (
    <Components.Page>
      <Components.Section>
        <Components.Layout>
          <Input.TextField name="name" label="Filter name" columns={6} placeholder="eg. Filter 1" />

          {!isLoading && (
            <Input.IndexerMultiSelect name="indexers" options={indexerOptions} label="Indexers" columns={6} />
          )}
        </Components.Layout>
      </Components.Section>

      <Components.Section
        title="Rules"
        subtitle="Specify rules on how torrents should be handled/selected."
      >
        <Components.Layout>
          <Input.TextField
            name="min_size"
            label="Min size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Input.TextField
            name="max_size"
            label="Max size"
            columns={6}
            placeholder="eg. 100MiB, 80GB"
            tooltip={
              <div>
                <p>Supports units such as MB, MiB, GB, etc.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Input.NumberField
            name="delay"
            label="Delay"
            placeholder="Number of seconds to delay actions"
            tooltip={
              <div>
                <p>Number of seconds to wait before running actions.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Input.NumberField
            name="priority"
            label="Priority"
            placeholder="Higher number = higher priority"
            tooltip={
              <div>
                <p>Filters are checked in order of priority. Higher number = higher priority.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Input.NumberField
            name="max_downloads"
            label="Max downloads"
            placeholder="Takes any number (0 is infinite)"
            tooltip={
              <div>
                <p>Number of max downloads as specified by the respective unit.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
          <Input.Select
            name="max_downloads_unit"
            label="Max downloads per"
            options={downloadsPerUnitOptions}
            optionDefaultText="Select unit"
            tooltip={
              <div>
                <p>The unit of time for counting the maximum downloads per filter.</p>
                <DocsLink href="https://autobrr.com/filters#rules" />
              </div>
            }
          />
        </Components.Layout>

        <Components.Layout>
          <Input.SwitchGroup
            name="enabled"
            label="Enabled"
            description="Enable or disable this filter."
            className="pb-2 col-span-12 sm:col-span-6"
          />
        </Components.Layout>
      </Components.Section>
    </Components.Page>
  );
};
