import { useMutation } from "react-query";
import { APIClient } from "../../api/APIClient";
import { queryClient } from "../../App";
import { toast } from "react-hot-toast";
import { Toast } from "../../components/notifications/Toast";
import { SlideOver } from "../../components/panels";
import { NumberFieldWide, PasswordFieldWide, SwitchGroupWide, TextFieldWide } from "../../components/inputs";
import { ImplementationMap } from "../../screens/settings/Feed";
import { componentMapType } from "./DownloadClientForms";

interface UpdateProps {
  isOpen: boolean;
  toggle: () => void;
  feed: Feed;
}

export function FeedUpdateForm({ isOpen, toggle, feed }: UpdateProps) {
  const mutation = useMutation(
    (feed: Feed) => APIClient.feeds.update(feed),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["feeds"]);
        toast.custom((t) => <Toast type="success" body={`${feed.name} was updated successfully`} t={t}/>);
        toggle();
      }
    }
  );

  const deleteMutation = useMutation(
    (feedID: number) => APIClient.feeds.delete(feedID),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(["feeds"]);
        toast.custom((t) => <Toast type="success" body={`${feed.name} was deleted.`} t={t}/>);
      }
    }
  );

  const onSubmit = (formData: unknown) => {
    mutation.mutate(formData as Feed);
  };

  const deleteAction = () => {
    deleteMutation.mutate(feed.id);
  };

  const initialValues = {
    id: feed.id,
    indexer: feed.indexer,
    enabled: feed.enabled,
    type: feed.type,
    name: feed.name,
    url: feed.url,
    api_key: feed.api_key,
    interval: feed.interval
  };

  return (
    <SlideOver
      type="UPDATE"
      title="Feed"
      isOpen={isOpen}
      toggle={toggle}
      onSubmit={onSubmit}
      deleteAction={deleteAction}
      initialValues={initialValues}
    >
      {(values) => (
        <div>
          <TextFieldWide name="name" label="Name" required={true}/>

          <div className="space-y-4 divide-y divide-gray-200 dark:divide-gray-700">
            <div
              className="py-4 flex items-center justify-between space-y-1 px-4 sm:space-y-0 sm:grid sm:grid-cols-3 sm:gap-4 sm:px-6 sm:py-5">
              <div>
                <label
                  htmlFor="type"
                  className="block text-sm font-medium text-gray-900 dark:text-white"
                >
                  Type
                </label>
              </div>
              <div className="flex justify-end sm:col-span-2">
                {ImplementationMap[feed.type]}
              </div>
            </div>

            <div className="py-6 px-6 space-y-6 sm:py-0 sm:space-y-0 sm:divide-y sm:divide-gray-200">
              <SwitchGroupWide name="enabled" label="Enabled"/>
            </div>
          </div>
          {componentMap[values.type]}
        </div>
      )}
    </SlideOver>
  );
}

function FormFieldsTorznab() {
  return (
    <div className="border-t border-gray-200 dark:border-gray-700 py-5">
      <TextFieldWide
        name="url"
        label="URL"
        help="Torznab url"
      />

      <PasswordFieldWide name="api_key" label="API key"/>

      <NumberFieldWide name="interval" label="Refresh interval"
        help="Minutes. Recommended 15-30. To low and risk ban."/>
    </div>
  );
}

const componentMap: componentMapType = {
  TORZNAB: <FormFieldsTorznab/>
};