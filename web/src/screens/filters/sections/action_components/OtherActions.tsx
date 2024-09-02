/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { WarningAlert } from "@components/alerts";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, NumberField, TextAreaAutoResize, TextField } from "@components/inputs";


export const SABnzbd = ({ idx, action, clients }: ClientActionProps) => (
  <FilterSection
    title="Instance"
    subtitle={
      <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
    }
  >
    <FilterLayout>
      <FilterHalfRow>
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
      </FilterHalfRow>
      <FilterHalfRow>
        <TextField
          name={`actions.${idx}.category`}
          label="Category"
          columns={6}
          placeholder="eg. category"
          tooltip={<p>Category must exist already.</p>}
        />
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
);

export const Test = () => (
  <WarningAlert
    alert="Heads up!"
    className="mt-2"
    colors="text-fuchsia-700 bg-fuchsia-100 dark:bg-fuchsia-200 dark:text-fuchsia-800"
    text="The test action does nothing except to show if the filter works. Make sure to have your Logs page open while testing."
  />
);

export const Exec = ({ idx }: ClientActionProps) => (
  <FilterSection
    title="Exec Arguments"
    subtitle="Specify the executable and its arguments to be executed upon filter match. Use an absolute path."
  >
    <FilterLayout>
      <TextField
        name={`actions.${idx}.exec_cmd`}
        label="Path to Executable"
        placeholder="Path to program eg. /bin/test"
      />

      <TextAreaAutoResize
        name={`actions.${idx}.exec_args`}
        label="Arguments"
        placeholder="Arguments eg. --test"
      />
    </FilterLayout>

  </FilterSection>
);

export const WatchFolder = ({ idx }: ClientActionProps) => (
  <FilterSection
    title="Watch Folder Arguments"
    subtitle="Point to where autobrr should save the files it fetches. Use an absolute path."
  >
    <FilterLayout>
      <TextAreaAutoResize
        name={`actions.${idx}.watch_folder`}
        label="Watch directory"
        placeholder="Watch directory eg. /home/user/rwatch"
      />
    </FilterLayout>
  </FilterSection>
);

export const WebHook = ({ idx }: ClientActionProps) => (
  <FilterSection
    title="Webhook Arguments"
    subtitle="Specify the payload to be sent to the desired endpoint upon filter match."
  >
    <FilterLayout>
      <TextField
        name={`actions.${idx}.webhook_host`}
        label="Endpoint"
        columns={6}
        placeholder="Host eg. http://localhost/webhook"
        tooltip={
          <p>URL or IP to your API. Pass params and set API tokens etc.</p>
        }
      />
    </FilterLayout>
    <TextAreaAutoResize
      name={`actions.${idx}.webhook_data`}
      label="Payload (json)"
      placeholder={"Request data: { \"key\": \"value\" }"}
    />
  </FilterSection>
);

export const Arr = ({ idx, action, clients }: ClientActionProps) => (
  <FilterSection
    title="Instance"
    subtitle={
      <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
    }
  >
    <FilterLayout>
      <FilterHalfRow>
        <DownloadClientSelect
          name={`actions.${idx}.client_id`}
          action={action}
          clients={clients}
        />
      </FilterHalfRow>

      <FilterHalfRow>
        <div className="">
          <TextField
            name={`actions.${idx}.external_download_client`}
            label="Override download client name for arr"
            tooltip={
              <p>
                Override Download client name from the one set in Clients. Useful if you
                have multiple clients inside the arr.
              </p>
            }
          />
          <NumberField
            name={`actions.${idx}.external_download_client_id`}
            label="Override download client id for arr DEPRECATED"
            className="mt-4"
            tooltip={
              <p>
                Override Download client Id from the one set in Clients. Useful if you
                have multiple clients inside the arr.
              </p>
            }
          />
        </div>
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
);
