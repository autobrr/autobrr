import React, {Fragment, useRef, useState} from "react";
import {Dialog, Switch, Transition} from "@headlessui/react";
import {ChevronDownIcon, ChevronRightIcon, ExclamationIcon,} from '@heroicons/react/solid'
import {EmptyListState} from "../components/EmptyListState";

import {
    Link,
    NavLink,
    Route,
    Switch as RouteSwitch,
    useHistory,
    useLocation,
    useParams,
    useRouteMatch
} from "react-router-dom";
import {FilterActionList} from "../components/FilterActionList";
import {DownloadClient, Filter, Indexer} from "../domain/interfaces";
import {useToggle} from "../hooks/hooks";
import {useMutation, useQuery} from "react-query";
import {queryClient} from "../App";
import {CONTAINER_OPTIONS, CODECS_OPTIONS, RESOLUTION_OPTIONS, SOURCES_OPTIONS} from "../domain/constants";
import {Field, Form} from "react-final-form";
import {MultiSelectField, TextField} from "../components/inputs";
import DEBUG from "../components/debug";
import TitleSubtitle from "../components/headings/TitleSubtitle";
import { SwitchGroup } from "../components/inputs";
import {classNames} from "../styles/utils";
import { FilterAddForm, FilterActionAddForm} from "../forms";
import Select from "react-select";
import APIClient from "../api/APIClient";

const tabs = [
    {name: 'General', href: '', current: true},
    // { name: 'TV', href: 'tv', current: false },
    // { name: 'Movies', href: 'movies', current: false },
    {name: 'Movies and TV', href: 'movies-tv', current: false},
    // { name: 'P2P', href: 'p2p', current: false },
    {name: 'Advanced', href: 'advanced', current: false},
    {name: 'Actions', href: 'actions', current: false},
]

function TabNavLink({item, url}: any) {
    const location = useLocation();

    const {pathname} = location;
    const splitLocation = pathname.split("/");

    // we need to clean the / if it's a base root path
    let too = item.href ? `${url}/${item.href}` : url

    return (
        <NavLink
            key={item.name}
            to={too}
            exact={true}
            activeClassName="border-purple-500 text-purple-600"
            className={classNames(
                'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
            )}
            aria-current={splitLocation[2] === item.href ? 'page' : undefined}
        >
            {item.name}
        </NavLink>
    )
}

export function Filters() {
    const [createFilterIsOpen, toggleCreateFilter] = useToggle(false)

    const {isLoading, error, data} = useQuery<Filter[], Error>('filter', APIClient.filters.getAll,
        {
            refetchOnWindowFocus: false
        }
    );

    if (isLoading) {
        return null
    }

    if (error) return (<p>'An error has occurred: '</p>)

    return (
        <main className="-mt-48 ">
            <FilterAddForm isOpen={createFilterIsOpen} toggle={toggleCreateFilter}/>

            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex justify-between">
                    <h1 className="text-3xl font-bold text-white capitalize">Filters</h1>

                    <div className="flex-shrink-0">
                        <button
                            type="button"
                            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                            onClick={toggleCreateFilter}
                        >
                            Add new
                        </button>
                    </div>
                </div>
            </header>

            <div className="max-w-7xl mx-auto pb-12 px-4 sm:px-6 lg:px-8">
                <div className="bg-white rounded-lg shadow">
                    <div className="relative inset-0 py-3 px-3 sm:px-3 lg:px-3 h-full">
                        {data && data.length > 0 ? <FilterList filters={data}/> :
                            <EmptyListState text="No filters here.." buttonText="Add new" buttonOnClick={toggleCreateFilter}/>}
                    </div>
                </div>
            </div>
        </main>
    )
}

interface FilterListProps {
    filters: Filter[];
}

function FilterList({filters}: FilterListProps) {
    return (
        <div className="flex flex-col">
            <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
                <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
                    <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
                        <table className="min-w-full divide-y divide-gray-200">
                            <thead className="bg-gray-50">
                            <tr>
                                <th
                                    scope="col"
                                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                >
                                    Enabled
                                </th>
                                <th
                                    scope="col"
                                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                >
                                    Name
                                </th>
                                <th
                                    scope="col"
                                    className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                                >
                                    Indexers
                                </th>
                                <th scope="col" className="relative px-6 py-3">
                                    <span className="sr-only">Edit</span>
                                </th>
                            </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-gray-200">
                            {filters.map((filter: Filter, idx) => (
                                <FilterListItem filter={filter} key={idx} idx={idx}/>
                            ))}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    )
}

interface FilterListItemProps {
    filter: Filter;
    idx: number;
}

function FilterListItem({filter, idx}: FilterListItemProps) {
    const [enabled, setEnabled] = useState(filter.enabled)

    const toggleActive = (status: boolean) => {
        console.log(status)
        setEnabled(status)
        // call api
    }

    return (
        <tr key={filter.name}
            className={idx % 2 === 0 ? 'bg-white' : 'bg-gray-50'}>
            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                <Switch
                    checked={enabled}
                    onChange={toggleActive}
                    className={classNames(
                        enabled ? 'bg-teal-500' : 'bg-gray-200',
                        'relative inline-flex flex-shrink-0 h-6 w-11 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500'
                    )}
                >
                    <span className="sr-only">Use setting</span>
                    <span
                        aria-hidden="true"
                        className={classNames(
                            enabled ? 'translate-x-5' : 'translate-x-0',
                            'inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition ease-in-out duration-200'
                        )}
                    />
                </Switch>
            </td>
            <td className="px-6 py-4 w-full whitespace-nowrap text-sm font-medium text-gray-900">{filter.name}</td>
            <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{filter.indexers && filter.indexers.map(t =>
                <span key={t.id} className="mr-2 inline-flex items-center px-2.5 py-0.5 rounded-md text-sm font-medium bg-gray-100 text-gray-800">{t.name}</span>)}</td>
            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <Link to={`filters/${filter.id.toString()}`} className="text-indigo-600 hover:text-indigo-900">
                    Edit
                </Link>
            </td>
        </tr>
    )
}

const FormButtonsGroup = ({deleteAction}: any) => {
    const [deleteModalIsOpen, toggleDeleteModal] = useToggle(false)

    const cancelButtonRef = useRef(null)

    return (
        <div className="pt-6 divide-y divide-gray-200">

            <Transition.Root show={deleteModalIsOpen} as={Fragment}>
                <Dialog
                    as="div"
                    static
                    className="fixed z-10 inset-0 overflow-y-auto"
                    initialFocus={cancelButtonRef}
                    open={deleteModalIsOpen}
                    onClose={toggleDeleteModal}
                >
                    <div
                        className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                        <Transition.Child
                            as={Fragment}
                            enter="ease-out duration-300"
                            enterFrom="opacity-0"
                            enterTo="opacity-100"
                            leave="ease-in duration-200"
                            leaveFrom="opacity-100"
                            leaveTo="opacity-0"
                        >
                            <Dialog.Overlay className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"/>
                        </Transition.Child>

                        {/* This element is to trick the browser into centering the modal contents. */}
                        <span className="hidden sm:inline-block sm:align-middle sm:h-screen" aria-hidden="true">
            &#8203;
          </span>
                        <Transition.Child
                            as={Fragment}
                            enter="ease-out duration-300"
                            enterFrom="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
                            enterTo="opacity-100 translate-y-0 sm:scale-100"
                            leave="ease-in duration-200"
                            leaveFrom="opacity-100 translate-y-0 sm:scale-100"
                            leaveTo="opacity-0 translate-y-4 sm:translate-y-0 sm:scale-95"
                        >
                            <div
                                className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
                                <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                                    <div className="sm:flex sm:items-start">
                                        <div
                                            className="mx-auto flex-shrink-0 flex items-center justify-center h-12 w-12 rounded-full bg-red-100 sm:mx-0 sm:h-10 sm:w-10">
                                            <ExclamationIcon className="h-6 w-6 text-red-600" aria-hidden="true"/>
                                        </div>
                                        <div className="mt-3 text-center sm:mt-0 sm:ml-4 sm:text-left">
                                            <Dialog.Title as="h3"
                                                          className="text-lg leading-6 font-medium text-gray-900">
                                                Remove filter
                                            </Dialog.Title>
                                            <div className="mt-2">
                                                <p className="text-sm text-gray-500">
                                                    Are you sure you want to remove this filter?
                                                    This action cannot be undone.
                                                </p>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                                    <button
                                        type="button"
                                        className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-red-600 text-base font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:ml-3 sm:w-auto sm:text-sm"
                                        onClick={deleteAction}
                                    >
                                        Remove
                                    </button>
                                    <button
                                        type="button"
                                        className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                                        onClick={toggleDeleteModal}
                                        ref={cancelButtonRef}
                                    >
                                        Cancel
                                    </button>
                                </div>
                            </div>
                        </Transition.Child>
                    </div>
                </Dialog>
            </Transition.Root>

            <div className="mt-4 pt-4 flex justify-between">
                <button
                    type="button"
                    className="inline-flex items-center justify-center px-4 py-2 border border-transparent font-medium rounded-md text-red-700 bg-red-100 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500 sm:text-sm"
                    onClick={toggleDeleteModal}
                >
                    Remove
                </button>

                <div>
                    <button
                        type="button"
                        className="bg-white border border-gray-300 rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-light-blue-500"
                    >
                        Cancel
                    </button>
                    <button
                        type="submit"
                        className="ml-4 relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                    >
                        Save
                    </button>
                </div>
            </div>
        </div>
    )
}

export function FilterDetails() {
    let {url} = useRouteMatch();
    let history = useHistory();
    let {filterId}: any = useParams();

    const {isLoading, data} = useQuery<Filter, Error>(['filter', parseInt(filterId)], () => APIClient.filters.getByID(parseInt(filterId)),
        {
            retry: false,
            refetchOnWindowFocus: false,
            onError: err => {
                history.push("./")
            }
        },
    )

    if (isLoading) {
        return null
    }

    if (!data) {
        return null
    }

    return (
        <main className="-mt-48 ">
            <header className="py-10">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center">
                    <h1 className="text-3xl font-bold text-white capitalize">
                        <NavLink to="/filters" exact={true}>
                            Filters
                        </NavLink>
                    </h1>
                    <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>
                    <h1 className="text-3xl font-bold text-white capitalize">{data.name}</h1>
                </div>
            </header>
            <div className="max-w-7xl mx-auto pb-12 px-4 sm:px-6 lg:px-8">
                <div className="bg-white rounded-lg shadow">
                    <div className="relative mx-auto md:px-6 xl:px-4">
                        <div className="px-4 sm:px-6 md:px-0">
                            <div className="py-6">
                                {/* Tabs */}
                                <div className="lg:hidden">
                                    <label htmlFor="selected-tab" className="sr-only">
                                        Select a tab
                                    </label>
                                    <select
                                        id="selected-tab"
                                        name="selected-tab"
                                        className="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-purple-500 focus:border-purple-500 sm:text-sm rounded-md"
                                    >
                                        {tabs.map((tab) => (
                                            <option key={tab.name}>{tab.name}</option>
                                        ))}
                                    </select>
                                </div>
                                <div className="hidden lg:block">
                                    <div className="border-b border-gray-200">
                                        <nav className="-mb-px flex space-x-8">
                                            {tabs.map((tab) => (
                                                <TabNavLink item={tab} url={url} key={tab.href}/>
                                            ))}
                                        </nav>
                                    </div>
                                </div>

                                <RouteSwitch>
                                    <Route exact path={url}>
                                        <FilterTabGeneral filter={data}/>
                                    </Route>

                                    <Route path={`${url}/movies-tv`}>
                                        {/*<FilterTabMoviesTv filter={data}/>*/}
                                        <FilterTabMoviesTvNew2 filter={data}/>
                                    </Route>

                                    {/*<Route path={`${path}/movies`}>*/}
                                    {/*    <p>movies</p>*/}
                                    {/*</Route>*/}

                                    {/*<Route path={`${path}/p2p`}>*/}
                                    {/*    <p>p2p</p>*/}
                                    {/*</Route>*/}

                                    <Route path={`${url}/advanced`}>
                                        <FilterTabAdvanced filter={data}/>
                                    </Route>

                                    <Route path={`${url}/actions`}>
                                        <FilterTabActions filter={data}/>
                                    </Route>

                                </RouteSwitch>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </main>
    )
}

interface FilterTabGeneralProps {
    filter: Filter;
}

function FilterTabGeneral({filter}: FilterTabGeneralProps) {
    const history = useHistory();

    const { data } = useQuery<Indexer[], Error>('indexerList', APIClient.indexers.getOptions,
        {
            refetchOnWindowFocus: false
        }
    )

    const updateMutation = useMutation((filter: Filter) => APIClient.filters.update(filter), {
        onSuccess: () => {
            // queryClient.setQueryData(['filter', filter.id], data)
            queryClient.invalidateQueries(["filter",filter.id]);
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.filters.delete(id), {
        onSuccess: () => {
            // invalidate filters
            queryClient.invalidateQueries("filter");
            // redirect
            history.push("/filters")
        }
    })

    const submitOther = (data: Filter) => {
        updateMutation.mutate(data)
    }

    const deleteAction = () => {
        deleteMutation.mutate(filter.id)
    }

    return (
        <div>
            <Form
                initialValues={{
                    id: filter.id,
                    name: filter.name,
                    enabled: filter.enabled,
                    min_size: filter.min_size,
                    max_size: filter.max_size,
                    delay: filter.delay,
                    shows: filter.shows,
                    years: filter.years,
                    resolutions: filter.resolutions || [],
                    sources: filter.sources || [],
                    codecs: filter.codecs || [],
                    containers: filter.containers || [],
                    seasons: filter.seasons,
                    episodes: filter.episodes,
                    match_releases: filter.match_releases,
                    except_releases: filter.except_releases,
                    match_release_groups: filter.match_release_groups,
                    except_release_groups: filter.except_release_groups,
                    match_categories: filter.match_categories,
                    except_categories: filter.except_categories,
                    match_tags: filter.match_tags,
                    except_tags: filter.except_tags,
                    match_uploaders: filter.match_uploaders,
                    except_uploaders: filter.except_uploaders,
                    freeleech: filter.freeleech,
                    freeleech_percent: filter.freeleech_percent,
                    indexers: filter.indexers || [],
                }}
                // validate={validate}
                onSubmit={submitOther}
            >
                {({handleSubmit, submitting, values, valid}) => {
                    return (
                        <form onSubmit={handleSubmit}>
                            <div>
                                <div className="mt-6 lg:pb-8">

                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="name" label="Filter name" columns={6} placeholder="eg. Filter 1"/>

                                        <div className="col-span-6">
                                            <label htmlFor="indexers" className="block text-xs font-bold text-gray-700 uppercase tracking-wide">
                                                Indexers
                                            </label>
                                                <Field
                                                    name="indexers"
                                                    multiple={true}
                                                    parse={val => val && val.map((item: any) => ({ id: item.value.id, name: item.value.name, enabled: item.value.enabled, identifier: item.value.identifier}))}
                                                    format={values => values.map((val: any) => ({ label: val.name, value: val}))}
                                                    render={({input, meta}) => (
                                                                <Select {...input}
                                                                        isClearable={true}
                                                                        isMulti={true}
                                                                        placeholder="Choose indexers"
                                                                        className="mt-2 block w-full focus:outline-none focus:ring-light-blue-500 focus:border-light-blue-500 sm:text-sm"
                                                                        options={data ? data.map(v => ({
                                                                            label: v.name,
                                                                            value: v
                                                                        })) : []}
                                                                />
                                                        )}
                                                />
                                        </div>
                                    </div>
                                </div>

                                <div className="mt-6 lg:pb-8">
                                    <TitleSubtitle title="Rules" subtitle="Set rules"/>

                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="min_size" label="Min size" columns={6} placeholder=""/>
                                        <TextField name="max_size" label="Max size" columns={6} placeholder=""/>
                                        <TextField name="delay" label="Delay" columns={6} placeholder=""/>
                                    </div>
                                </div>

                                <div className="border-t">
                                    <SwitchGroup name="enabled" label="Enabled" description="Enabled or disable filter."/>
                                </div>

                            </div>

                            <FormButtonsGroup deleteAction={deleteAction}/>

                            <DEBUG values={values}/>
                        </form>
                    )
                }}
            </Form>
        </div>
    );
}


function FilterTabMoviesTvNew2({filter}: FilterTabGeneralProps) {
    const history = useHistory();

    const updateMutation = useMutation((filter: Filter) => APIClient.filters.update(filter), {
        onSuccess: () => {
            // queryClient.setQueryData(['filter', filter.id], data)
            queryClient.invalidateQueries(["filter",filter.id]);
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.filters.delete(id), {
        onSuccess: () => {
            // invalidate filters
            queryClient.invalidateQueries("filter");
            // redirect
            history.push("/filters")
        }
    })

    const deleteAction = () => {
        deleteMutation.mutate(filter.id)
    }

    const submitOther = (data: Filter) => {
        updateMutation.mutate(data)
    }

    return (
        <div>
            <Form
                initialValues={{
                    id: filter.id,
                    name: filter.name,
                    enabled: filter.enabled,
                    min_size: filter.min_size,
                    max_size: filter.max_size,
                    delay: filter.delay,
                    shows: filter.shows,
                    years: filter.years,
                    resolutions: filter.resolutions || [],
                    sources: filter.sources || [],
                    codecs: filter.codecs || [],
                    containers: filter.containers || [],
                    seasons: filter.seasons,
                    episodes: filter.episodes,
                    match_releases: filter.match_releases,
                    except_releases: filter.except_releases,
                    match_release_groups: filter.match_release_groups,
                    except_release_groups: filter.except_release_groups,
                    match_categories: filter.match_categories,
                    except_categories: filter.except_categories,
                    match_tags: filter.match_tags,
                    except_tags: filter.except_tags,
                    match_uploaders: filter.match_uploaders,
                    except_uploaders: filter.except_uploaders,
                    freeleech: filter.freeleech,
                    freeleech_percent: filter.freeleech_percent,
                    indexers: filter.indexers || [],
                }}
                // validate={validate}
                onSubmit={submitOther}
            >
                {({handleSubmit, submitting, values, valid}) => {
                    return (
                        <form onSubmit={handleSubmit}>
                            <div className="mt-6 grid grid-cols-12 gap-6">
                                <TextField name="shows" label="Movies / Shows" columns={8} placeholder="eg. Movie, Show 1, Show?2"/>
                                <TextField name="years" label="Years" columns={4} placeholder="eg. 2018,2019-2021"/>
                            </div>

                            <div className="mt-6 lg:pb-8">
                                <TitleSubtitle title="Seasons and Episodes" subtitle="Set seaons and episodes"/>

                                <div className="mt-6 grid grid-cols-12 gap-6">
                                    <TextField name="seasons" label="Seasons" columns={8} placeholder="eg. 1, 3, 2-6"/>
                                    <TextField name="episodes" label="Episodes" columns={4} placeholder="eg. 2, 4, 10-20"/>
                                </div>
                            </div>

                            <div className="mt-6 lg:pb-8">
                                <TitleSubtitle title="Quality" subtitle="Resolution, source etc."/>

                                <div className="mt-6 grid grid-cols-12 gap-6">
                                    <MultiSelectField name="resolutions" options={RESOLUTION_OPTIONS} label="resolutions" columns={6}/>
                                    <MultiSelectField name="sources" options={SOURCES_OPTIONS} label="sources" columns={6}/>
                                </div>

                                <div className="mt-6 grid grid-cols-12 gap-6">
                                    <MultiSelectField name="codecs" options={CODECS_OPTIONS} label="codecs" columns={6}/>
                                    <MultiSelectField name="containers" options={CONTAINER_OPTIONS} label="containers" columns={6}/>
                                </div>
                            </div>

                            <FormButtonsGroup deleteAction={deleteAction}/>

                            <DEBUG values={values}/>
                        </form>
                    )
                }}
            </Form>
        </div>
    )
}

function FilterTabAdvanced({filter}: FilterTabGeneralProps) {
    const history = useHistory();
    const [releasesIsOpen, toggleReleases] = useToggle(false)
    const [groupsIsOpen, toggleGroups] = useToggle(false)
    const [categoriesIsOpen, toggleCategories] = useToggle(false)
    const [uploadersIsOpen, toggleUploaders] = useToggle(false)
    const [freeleechIsOpen, toggleFreeleech] = useToggle(false)

    const updateMutation = useMutation((filter: Filter) => APIClient.filters.update(filter), {
        onSuccess: () => {
            // queryClient.setQueryData(['filter', filter.id], data)
            queryClient.invalidateQueries(["filter",filter.id]);
        }
    })

    const deleteMutation = useMutation((id: number) => APIClient.filters.delete(id), {
        onSuccess: () => {
            // invalidate filters
            queryClient.invalidateQueries("filter");
            // redirect
            history.push("/filters")
        }
    })

    const deleteAction = () => {
        deleteMutation.mutate(filter.id)
    }

    const submitOther = (data: Filter) => {
        updateMutation.mutate(data)
    }

    return (
        <div>
            <Form
                initialValues={{
                    id: filter.id,
                    name: filter.name,
                    enabled: filter.enabled,
                    min_size: filter.min_size,
                    max_size: filter.max_size,
                    delay: filter.delay,
                    shows: filter.shows,
                    years: filter.years,
                    resolutions: filter.resolutions || [],
                    sources: filter.sources || [],
                    codecs: filter.codecs || [],
                    containers: filter.containers || [],
                    seasons: filter.seasons,
                    episodes: filter.episodes,
                    match_releases: filter.match_releases,
                    except_releases: filter.except_releases,
                    match_release_groups: filter.match_release_groups,
                    except_release_groups: filter.except_release_groups,
                    match_categories: filter.match_categories,
                    except_categories: filter.except_categories,
                    match_tags: filter.match_tags,
                    except_tags: filter.except_tags,
                    match_uploaders: filter.match_uploaders,
                    except_uploaders: filter.except_uploaders,
                    freeleech: filter.freeleech,
                    freeleech_percent: filter.freeleech_percent,
                    indexers: filter.indexers || [],
                }}
                // validate={validate}
                onSubmit={submitOther}
            >
                {({handleSubmit, submitting, values, valid}) => {
                    return (
                        <form onSubmit={handleSubmit}>
                            <div className="mt-6 lg:pb-8 border-b border-gray-200">
                                <div className="flex justify-between items-center cursor-pointer" onClick={toggleReleases}>
                                    <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
                                        <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900">Releases</h3>
                                        <p className="ml-2 mt-1 text-sm text-gray-500 truncate">Match or ignore</p>
                                    </div>
                                    <div className="mt-3 sm:mt-0 sm:ml-4">
                                        <button
                                            type="button"
                                            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
                                        >
                                            {releasesIsOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/> :  <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>}
                                        </button>
                                    </div>
                                </div>
                                {releasesIsOpen && (
                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="match_releases" label="Match releases" columns={6} placeholder=""/>
                                        <TextField name="except_releases" label="Except releases" columns={6}
                                                   placeholder=""/>
                                    </div>
                                )}
                            </div>

                            <div className="mt-6 lg:pb-8 border-b border-gray-200">
                                <div className="flex justify-between items-center cursor-pointer" onClick={toggleGroups}>
                                    <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
                                        <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900">Groups</h3>
                                        <p className="ml-2 mt-1 text-sm text-gray-500 truncate">Match or ignore</p>
                                    </div>
                                    <div className="mt-3 sm:mt-0 sm:ml-4">
                                        <button
                                            type="button"
                                            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
                                        >
                                            {groupsIsOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/> :  <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>}
                                        </button>
                                    </div>
                                </div>
                                {groupsIsOpen && (
                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="match_releases" label="Match releases" columns={6} placeholder=""/>
                                        <TextField name="except_releases" label="Except releases" columns={6}
                                                   placeholder=""/>
                                    </div>
                                )}
                            </div>

                            <div className="mt-6 lg:pb-8 border-b border-gray-200">
                                <div className="flex justify-between items-center cursor-pointer" onClick={toggleCategories}>
                                    <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
                                        <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900">Categories and tags</h3>
                                        <p className="ml-2 mt-1 text-sm text-gray-500 truncate">Match or ignore categories or tags</p>
                                    </div>
                                    <div className="mt-3 sm:mt-0 sm:ml-4">
                                        <button
                                            type="button"
                                            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
                                        >
                                            {categoriesIsOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/> :  <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>}
                                        </button>
                                    </div>
                                </div>
                                {categoriesIsOpen && (
                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="match_categories" label="Match categories" columns={6}
                                                   placeholder=""/>
                                        <TextField name="except_categories" label="Except categories" columns={6}
                                                   placeholder=""/>

                                        <TextField name="match_tags" label="Match tags" columns={6} placeholder=""/>
                                        <TextField name="except_tags" label="Except tags" columns={6} placeholder=""/>
                                    </div>
                                )}
                            </div>

                            <div className="mt-6 lg:pb-8 border-b border-gray-200">
                                <div className="flex justify-between items-center cursor-pointer" onClick={toggleUploaders}>
                                    <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
                                        <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900">Uploaders</h3>
                                        <p className="ml-2 mt-1 text-sm text-gray-500 truncate">Match or ignore uploaders</p>
                                    </div>
                                    <div className="mt-3 sm:mt-0 sm:ml-4">
                                        <button
                                            type="button"
                                            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
                                        >
                                            {uploadersIsOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/> :  <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>}
                                        </button>
                                    </div>
                                </div>
                                {uploadersIsOpen && (
                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <TextField name="match_uploaders" label="Match uploaders" columns={6}
                                                   placeholder=""/>
                                        <TextField name="except_uploaders" label="Except uploaders" columns={6}
                                                   placeholder=""/>
                                    </div>
                                )}
                            </div>

                            <div className="mt-6 lg:pb-8 border-b border-gray-200">
                                <div className="flex justify-between items-center cursor-pointer" onClick={toggleFreeleech}>
                                    <div className="-ml-2 -mt-2 flex flex-wrap items-baseline">
                                        <h3 className="ml-2 mt-2 text-lg leading-6 font-medium text-gray-900">Freeleech</h3>
                                        <p className="ml-2 mt-1 text-sm text-gray-500 truncate">Match only freeleech and freeleech percent</p>
                                    </div>
                                    <div className="mt-3 sm:mt-0 sm:ml-4">
                                        <button
                                            type="button"
                                            className="inline-flex items-center px-4 py-2 border-transparent text-sm font-medium text-white"
                                        >
                                            {freeleechIsOpen ? <ChevronDownIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/> :  <ChevronRightIcon className="h-6 w-6 text-gray-500" aria-hidden="true"/>}
                                        </button>
                                    </div>
                                </div>
                                {freeleechIsOpen && (
                                    <div className="mt-6 grid grid-cols-12 gap-6">
                                        <div className="col-span-6">
                                            <SwitchGroup name="freeleech" label="Freeleech" />
                                        </div>

                                        <TextField name="freeleech_percent" label="Freeleech percent" columns={6} />
                                    </div>
                                )}
                            </div>

                            <FormButtonsGroup deleteAction={deleteAction}/>

                            <DEBUG values={values}/>

                        </form>
                    )
                }}
            </Form>
        </div>
    )
}

function FilterTabActions({filter}: FilterTabGeneralProps) {
    const [addActionIsOpen, toggleAddAction] = useToggle(false)

    const {data} = useQuery<DownloadClient[], Error>('downloadClients', APIClient.download_clients.getAll,
        {
            refetchOnWindowFocus: false
        }
    )

    return (
        <div className="mt-10">
            {addActionIsOpen &&
            <FilterActionAddForm filter={filter} clients={data || []} isOpen={addActionIsOpen} toggle={toggleAddAction}/>
            }
            <div>
                <div className="-ml-4 -mt-4 mb-6 flex justify-between items-center flex-wrap sm:flex-nowrap">
                    <div className="ml-4 mt-4">
                        <h3 className="text-lg leading-6 font-medium text-gray-900">Actions</h3>
                        <p className="mt-1 text-sm text-gray-500">
                            Actions
                        </p>
                    </div>
                    <div className="ml-4 mt-4 flex-shrink-0">
                        <button
                            type="button"
                            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
                            onClick={toggleAddAction}
                        >
                            Add new
                        </button>
                    </div>
                </div>
                {filter.actions ? <FilterActionList actions={filter.actions} clients={data || []} filterID={filter.id}/> :
                    <EmptyListState text="No actions yet!"/>}
            </div>
        </div>
    )
}


