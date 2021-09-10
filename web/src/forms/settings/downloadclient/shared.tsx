import { Fragment } from "react";
import { SwitchGroup, TextFieldWide } from "../../../components/inputs";
import { NumberFieldWide, PasswordFieldWide } from "../../../components/inputs/wide";
import { useField } from "react-final-form";
import { Dialog } from "@headlessui/react";

function FormFieldsDefault() {
  return (
    <Fragment>
      <TextFieldWide name="host" label="Host" help="Url domain.ltd/client" />

      <NumberFieldWide name="port" label="Port" />

      <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
        <SwitchGroup name="ssl" label="SSL" />
      </div>

      <TextFieldWide name="username" label="Username" />
      <PasswordFieldWide name="password" label="Password" />
    </Fragment>
  );
}

function FormFieldsArr() {
  const { input } = useField("settings.basic.auth");
  return (
    <Fragment>
      <TextFieldWide name="host" label="Host" help="Full url like http(s)://domain.ltd/" />

      <PasswordFieldWide name="settings.apikey" label="API key" />

      <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
        <SwitchGroup name="settings.basic.auth" label="Basic auth" />
      </div>

      {input.value === true && (
        <Fragment>
          <TextFieldWide name="settings.basic.username" label="Username" />
          <PasswordFieldWide name="settings.basic.password" label="Password" />
        </Fragment>
      )}
    </Fragment>
  );
}

export const componentMap: any = {
  DELUGE_V1: <FormFieldsDefault />,
  DELUGE_V2: <FormFieldsDefault />,
  QBITTORRENT: <FormFieldsDefault />,
  RADARR: <FormFieldsArr />,
  SONARR: <FormFieldsArr />,
  LIDARR: <FormFieldsArr />,
};


function FormFieldsRulesBasic() {
  const { input: enabled } = useField("settings.rules.enabled");

  return (
    <div className="border-t border-gray-200 py-5">

      <div className="px-6 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900">Rules</Dialog.Title>
        <p className="text-sm text-gray-500">
          Manage max downloads.
        </p>
      </div>

      <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
        <SwitchGroup name="settings.rules.enabled" label="Enabled" />
      </div>

      {enabled.value === true && (
        <Fragment>
          <NumberFieldWide name="settings.rules.max_active_downloads" label="Max active downloads" />
        </Fragment>
      )}
    </div>
  );
}

function FormFieldsRules() {
  const { input } = useField("settings.rules.ignore_slow_torrents");
  const { input: enabled } = useField("settings.rules.enabled");

  return (
    <div className="border-t border-gray-200 py-5">

      <div className="px-6 space-y-1">
        <Dialog.Title className="text-lg font-medium text-gray-900">Rules</Dialog.Title>
        <p className="text-sm text-gray-500">
          Manage max downloads etc.
        </p>
      </div>

      <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
        <SwitchGroup name="settings.rules.enabled" label="Enabled" />
      </div>

      {enabled.value === true && (
        <Fragment>
          <NumberFieldWide name="settings.rules.max_active_downloads" label="Max active downloads" />
          <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
            <SwitchGroup name="settings.rules.ignore_slow_torrents" label="Ignore slow torrents" />
          </div>

          {input.value === true && (
            <Fragment>
              <NumberFieldWide name="settings.rules.download_speed_threshold" label="Download speed threshold" placeholder="in KB/s" help="If download speed is below this when max active downloads is hit, download anyways. KB/s" />
            </Fragment>
          )}
        </Fragment>
      )}
    </div>
  );
}

export const rulesComponentMap: any = {
  DELUGE_V1: <FormFieldsRulesBasic />,
  DELUGE_V2: <FormFieldsRulesBasic />,
  QBITTORRENT: <FormFieldsRules />,
};