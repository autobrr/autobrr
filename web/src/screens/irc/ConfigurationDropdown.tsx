import { Fragment } from "react";
import { Menu, Transition } from "@headlessui/react";
import { Cog6ToothIcon } from "@heroicons/react/24/solid";

import { Checkbox } from "@components/Checkbox";
import { SettingsContext } from "@utils/Context";


export const ConfigurationDropdown = () => {
  const [settings, setSettings] = SettingsContext.use();

  const onSetValue = (
    key: "scrollOnNewLog",
    newValue: boolean
  ) => setSettings((prevState) => ({
    ...prevState,
    [key]: newValue
  }));

  //
  // FIXME: Warning: Function components cannot be given refs. Attempts to access this ref will fail.
  //        Did you mean to use React.forwardRef()?
  //
  // Check the render method of `Pe2`.
  //  at Checkbox (http://localhost:3000/src/components/Checkbox.tsx:14:28)
  //  at Pe2 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:2164:12)
  //  at div
  //  at Ee (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:2106:12)
  //  at c5 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:592:22)
  //  at De4 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:3016:22)
  //  at He5 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:3053:15)
  //  at div
  //  at c5 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:592:22)
  //  at Me2 (http://localhost:3000/node_modules/.vite/deps/@headlessui_react.js?v=e8629745:2062:21)
  //  at IRCLogsDropdown (http://localhost:3000/src/screens/settings/Irc.tsx?t=1694269937935:1354:53)
  return (
    <Menu as="div" className="relative">
      <Menu.Button className="flex items-center text-gray-800 dark:text-gray-400 p-2 rounded border transition border-gray-400 dark:border-gray-800 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600">
        <Cog6ToothIcon className="w-5 h-5" title="Configure behavior" />
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items
          className="absolute z-10 right-0 mt-2 px-3 py-2 bg-gray-100 dark:bg-gray-900 border border-gray-400 dark:border-gray-700 rounded-md focus:outline-none"
        >
          <Menu.Item>
            {() => (
              <Checkbox
                label="Scroll to bottom on new message"
                value={settings.scrollOnNewLog}
                setValue={(newValue) => onSetValue("scrollOnNewLog", newValue)}
              />
            )}
          </Menu.Item>
        </Menu.Items>
      </Transition>
    </Menu>
  );
};
