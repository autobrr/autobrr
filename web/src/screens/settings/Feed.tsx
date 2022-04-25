import { useToggle } from "../../hooks/hooks";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { APIClient } from "../../api/APIClient";
import { Menu, Switch, Transition } from "@headlessui/react";

import type {FieldProps} from "formik";
import {classNames} from "../../utils";
import {Fragment, useRef, useState} from "react";
import {toast} from "react-hot-toast";
import Toast from "../../components/notifications/Toast";
import {queryClient} from "../../App";
import {DeleteModal} from "../../components/modals";
import {
    DotsHorizontalIcon,
    PencilAltIcon,
    SwitchHorizontalIcon,
    TrashIcon
} from "@heroicons/react/outline";
import {FeedUpdateForm} from "../../forms/settings/FeedForms";
import {EmptyBasic} from "../../components/emptystates";

function FeedSettings() {
    const {data} = useQuery<Feed[], Error>('feeds', APIClient.feeds.find,
        {
            refetchOnWindowFocus: false
        }
    )

    return (
        <div className="divide-y divide-gray-200 lg:col-span-9">
            <div className="py-6 px-4 sm:p-6 lg:pb-8">
                <div className="-ml-4 -mt-4 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900 dark:text-white">Feeds</h3>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                            Manage torznab feeds.
                        </p>
                    </div>
                </div>

                {data && data.length > 0 ?
                    <section className="mt-6 light:bg-white dark:bg-gray-800 light:shadow sm:rounded-md">
                        <ol className="min-w-full relative">
                            <li className="grid grid-cols-12 gap-4 border-b border-gray-200 dark:border-gray-700">
                                <div
                                    className="col-span-2 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Enabled
                                </div>
                                <div
                                    className="col-span-6 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Name
                                </div>
                                <div
                                    className="col-span-2 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Type
                                </div>
                                {/*<div className="col-span-4 px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">Events</div>*/}
                            </li>

                            {data && data.map((f) => (
                                <ListItem key={f.id} feed={f}/>
                            ))}
                        </ol>
                    </section>
                    : <EmptyBasic title="No feeds" subtitle="Setup via indexers" />}
            </div>
        </div>
    )
}

const ImplementationTorznab = () => (
    <span
        className="inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-orange-200 dark:bg-orange-400 text-orange-800 dark:text-orange-800"
    >
        Torznab
    </span>
)

export const ImplementationMap: any = {
    "TORZNAB": <ImplementationTorznab/>,
};

interface ListItemProps {
    feed: Feed;
}

function ListItem({feed}: ListItemProps) {
    const [updateFormIsOpen, toggleUpdateForm] = useToggle(false)

    const [enabled, setEnabled] = useState(feed.enabled)

    const updateMutation = useMutation(
        (status: boolean) => APIClient.feeds.toggleEnable(feed.id, status),
        {
            onSuccess: () => {
                toast.custom((t) => <Toast type="success"
                                           body={`${feed.name} was ${enabled ? "disabled" : "enabled"} successfully`}
                                           t={t}/>)

                queryClient.invalidateQueries(["feeds"]);
                queryClient.invalidateQueries(["feeds", feed?.id]);
            }
        }
    );

    const toggleActive = (status: boolean) => {
        setEnabled(status);
        updateMutation.mutate(status);
    }

    return (
        <li key={feed.id} className="text-gray-500 dark:text-gray-400">
            <FeedUpdateForm isOpen={updateFormIsOpen} toggle={toggleUpdateForm} feed={feed}/>

            <div className="grid grid-cols-12 gap-4 items-center py-4">
                <div className="col-span-2 flex items-center sm:px-6 ">
                    <Switch
                        checked={feed.enabled}
                        onChange={toggleActive}
                        className={classNames(
                            feed.enabled ? 'bg-teal-500 dark:bg-blue-500' : 'bg-gray-200 dark:bg-gray-600',
                            'relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500'
                        )}
                    >
                        <span className="sr-only">Use setting</span>
                        <span
                            aria-hidden="true"
                            className={classNames(
                                feed.enabled ? 'translate-x-5' : 'translate-x-0',
                                'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                            )}
                        />
                    </Switch>
                </div>
                <div className="col-span-6 flex items-center sm:px-6 text-sm font-medium text-gray-900 dark:text-white">
                    {feed.name}
                </div>
                <div className="col-span-2 flex items-center sm:px-6">
                    {ImplementationMap[feed.type]}
                </div>
                <div className="col-span-1 flex items-center sm:px-6">
                    <FeedItemDropdown
                        feed={feed}
                        onToggle={toggleActive}
                        toggleUpdate={toggleUpdateForm}
                    />
                </div>
            </div>
        </li>
    )
}

interface FeedItemDropdownProps {
    feed: Feed;
    onToggle: (newState: boolean) => void;
    toggleUpdate: () => void;
}

const FeedItemDropdown = ({
                              feed,
                              onToggle,
                              toggleUpdate,
                          }: FeedItemDropdownProps) => {
    const cancelModalButtonRef = useRef(null);

    const queryClient = useQueryClient();

    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false);
    const deleteMutation = useMutation(
        (id: number) => APIClient.feeds.delete(id),
        {
            onSuccess: () => {
                queryClient.invalidateQueries(["feeds"]);
                queryClient.invalidateQueries(["feeds", feed.id]);

                toast.custom((t) => <Toast type="success" body={`Feed ${feed?.name} was deleted`} t={t}/>);
            }
        }
    );

    return (
        <Menu as="div">
            <DeleteModal
                isOpen={deleteModalIsOpen}
                toggle={toggleDeleteModal}
                buttonRef={cancelModalButtonRef}
                deleteAction={() => {
                    deleteMutation.mutate(feed.id);
                    toggleDeleteModal();
                }}
                title={`Remove feed: ${feed.name}`}
                text="Are you sure you want to remove this feed? This action cannot be undone."
            />
            <Menu.Button className="px-4 py-2">
                <DotsHorizontalIcon
                    className="w-5 h-5 text-gray-700 hover:text-gray-900 dark:text-gray-100 dark:hover:text-gray-400"
                    aria-hidden="true"
                />
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
                    className="absolute right-0 w-56 mt-2 origin-top-right bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700 rounded-md shadow-lg ring-1 ring-black ring-opacity-10 focus:outline-none"
                >
                    <div className="px-1 py-1">
                        <Menu.Item>
                            {({active}) => (
                                <button
                                    className={classNames(
                                        active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                                        "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                                    )}
                                    onClick={() => toggleUpdate()}
                                >
                                    <PencilAltIcon
                                        className={classNames(
                                            active ? "text-white" : "text-blue-500",
                                            "w-5 h-5 mr-2"
                                        )}
                                        aria-hidden="true"
                                    />
                                    Edit
                                </button>
                            )}
                        </Menu.Item>
                        <Menu.Item>
                            {({active}) => (
                                <button
                                    className={classNames(
                                        active ? "bg-blue-600 text-white" : "text-gray-900 dark:text-gray-300",
                                        "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                                    )}
                                    onClick={() => onToggle(!feed.enabled)}
                                >
                                    <SwitchHorizontalIcon
                                        className={classNames(
                                            active ? "text-white" : "text-blue-500",
                                            "w-5 h-5 mr-2"
                                        )}
                                        aria-hidden="true"
                                    />
                                    Toggle
                                </button>
                            )}
                        </Menu.Item>
                    </div>
                    <div className="px-1 py-1">
                        <Menu.Item>
                            {({active}) => (
                                <button
                                    className={classNames(
                                        active ? "bg-red-600 text-white" : "text-gray-900 dark:text-gray-300",
                                        "font-medium group flex rounded-md items-center w-full px-2 py-2 text-sm"
                                    )}
                                    onClick={() => toggleDeleteModal()}
                                >
                                    <TrashIcon
                                        className={classNames(
                                            active ? "text-white" : "text-red-500",
                                            "w-5 h-5 mr-2"
                                        )}
                                        aria-hidden="true"
                                    />
                                    Delete
                                </button>
                            )}
                        </Menu.Item>
                    </div>
                </Menu.Items>
            </Transition>
        </Menu>
    );
}

export default FeedSettings;