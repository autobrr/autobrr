/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { WarningAlert } from "@components/alerts";
import { FilterHalfRow, FilterLayout, FilterSection } from "@screens/filters/sections/_components.tsx";
import { DownloadClientSelect, NumberField, TextAreaAutoResize, TextField } from "@components/inputs/tanstack";
import { ContextField } from "@app/lib/form";


export const SABnzbd = ({ idx, action, clients }: ClientActionProps) => (
  <FilterSection
    title="Instance"
    subtitle={
      <>Select the <span className="font-bold">specific instance</span> which you want to handle this release filter.</>
    }
  >
    <FilterLayout>
      <FilterHalfRow>
        <ContextField name={`actions.${idx}.client_id`}>
          <DownloadClientSelect
            action={action}
            clients={clients}
          />
        </ContextField>
      </FilterHalfRow>
      <FilterHalfRow>
        <ContextField name={`actions.${idx}.category`}>
          <TextField
            label="Category"
            columns={6}
            placeholder="eg. category"
            tooltip={<p>Category must exist already.</p>}
          />
        </ContextField>
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
      <ContextField name={`actions.${idx}.exec_cmd`}>
        <TextField
          label="Path to Executable"
          placeholder="Path to program eg. /bin/test"
        />
      </ContextField>

      <ContextField name={`actions.${idx}.exec_args`}>
        <TextAreaAutoResize
          label="Arguments"
          placeholder="Arguments eg. --test"
        />
      </ContextField>
    </FilterLayout>

  </FilterSection>
);

export const WatchFolder = ({ idx }: ClientActionProps) => (
  <FilterSection
    title="Watch Folder Arguments"
    subtitle="Point to where autobrr should save the files it fetches. Use an absolute path."
  >
    <FilterLayout>
      <ContextField name={`actions.${idx}.watch_folder`}>
        <TextAreaAutoResize
          label="Watch directory"
          placeholder="Watch directory eg. /home/user/rwatch"
        />
      </ContextField>
    </FilterLayout>
  </FilterSection>
);

export const WebHook = ({ idx }: ClientActionProps) => (
  <FilterSection
    title="Webhook Arguments"
    subtitle="Specify the payload to be sent to the desired endpoint upon filter match."
  >
    <FilterLayout>
      <ContextField name={`actions.${idx}.webhook_host`}>
        <TextField
          label="Endpoint"
          columns={6}
          placeholder="Host eg. http://localhost/webhook"
          tooltip={
            <p>URL or IP to your API. Pass params and set API tokens etc.</p>
          }
        />
      </ContextField>
    </FilterLayout>
    <ContextField name={`actions.${idx}.webhook_data`}>
      <TextAreaAutoResize
        label="Payload (json)"
        placeholder={"Request data: { \"key\": \"value\" }"}
      />
    </ContextField>
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
        <ContextField name={`actions.${idx}.client_id`}>
          <DownloadClientSelect
            action={action}
            clients={clients}
          />
        </ContextField>
      </FilterHalfRow>

      <FilterHalfRow>
        <div className="">
          <ContextField name={`actions.${idx}.external_download_client`}>
            <TextField
              label="Override download client name for arr"
              tooltip={
                <p>
                  Override Download client name from the one set in Clients. Useful if you
                  have multiple clients inside the arr.
                </p>
              }
            />
          </ContextField>
          <ContextField name={`actions.${idx}.external_download_client_id`}>
            <NumberField
              label="Override download client id for arr DEPRECATED"
              className="mt-4"
              tooltip={
                <p>
                  Override Download client Id from the one set in Clients. Useful if you
                  have multiple clients inside the arr.
                </p>
              }
            />
          </ContextField>
        </div>
      </FilterHalfRow>
    </FilterLayout>
  </FilterSection>
);
