/*
 * Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { APIClient } from "@api/APIClient";
import { PlusIcon } from "@heroicons/react/24/solid";

const Omegabrr = () => {
  const [selectedFilter, setSelectedFilter] = useState<string>("");
  const [selectedClient, setSelectedClient] = useState<string>("");

  // Fetch filters for the "Filter" dropdown
  const { data: filters, isFetching: isFetchingFilters } = useQuery({
    queryKey: ["filters"],
    queryFn: APIClient.filters.getAll,
    refetchOnWindowFocus: false
  });

  const [sourceType, setSourceType] = useState("clients"); // 'clients' or 'lists'

  // Fetch clients for the "Client" dropdown
  const { data: clients, isFetching: isFetchingClients } = useQuery({
    queryKey: ["download_clients"],
    queryFn: async () => {
      const allClients = await APIClient.download_clients.getAll();
      // Filter the clients based on the specified types
      return allClients.filter(client =>
        ["RADARR", "SONARR", "LIDARR", "WHISPARR", "READARR"].includes(client.type)
      );
    },
    refetchOnWindowFocus: false
  });

  const listOptions = [
    { name: "Trakt - Popular TV", url: "https://api.autobrr.com/lists/trakt/popular-tv" },
    { name: "Trakt - Anticipated TV", url: "https://api.autobrr.com/lists/trakt/anticipated-tv" },
    { name: "Trakt - Upcoming Movies", url: "https://api.autobrr.com/lists/trakt/upcoming-movies" },
    { name: "Trakt - Upcoming Blu-ray", url: "https://api.autobrr.com/lists/trakt/upcoming-bluray" },
    { name: "Trakt - Stevenlu", url: "https://api.autobrr.com/lists/stevenlu" },
    { name: "MDBList - New Movies", url: "https://mdblist.com/lists/linaspurinis/new-movies/json" },
    { name: "Metacritic - Upcoming Albums", url: "https://api.autobrr.com/lists/metacritic/upcoming-albums" },
    { name: "Metacritic - New Albums", url: "https://api.autobrr.com/lists/metacritic/new-albums" }
  ];

  const [selectedList, setSelectedList] = useState(listOptions[0].url);

  // Set default values when data is fetched
  useEffect(() => {
    if (filters && filters.length > 0) {
      setSelectedFilter(filters[0].name); // Assuming filters have a 'name' property
    }
    if (clients && clients.length > 0) {
      setSelectedClient(clients[0].name); // Set the default selected client
    }
  }, [filters, clients]);



  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9">
      <div className="py-6 px-4 sm:p-6">
        <div>
          <h2 className="text-lg leading-6 font-bold text-gray-900 dark:text-white">Omegabrr</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Omegabrr transforms items monitored by arrs or lists into autobrr filters. Useful for automating your filters for monitored media or racing criteria.
          </p>
        </div>
      </div>
      <div className="px-6 py-4">
        <div className="mb-4 flex items-center">
          <button
            className={`mr-2 px-4 py-2 text-s font-bold rounded-md ${sourceType === "clients" ? "bg-blue-500 text-white" : "bg-gray-200 text-gray-700"}`}
            onClick={() => setSourceType("clients")}
          >
            Client
          </button>
          <button
            className={`px-4 py-2 text-s font-bold rounded-md ${sourceType === "lists" ? "bg-blue-500 text-white" : "bg-gray-200 text-gray-700"}`}
            onClick={() => setSourceType("lists")}
          >
            Lists
          </button>
          <p className="text-gray-600 dark:text-gray-300 text-sm pl-4">Choose between an *arr client or a list.</p>
        </div>
  
        {/* Conditional rendering based on the sourceType */}
        {sourceType === "clients" && (
          <div className="mb-4">
            {/* Client dropdown */}
            <label htmlFor="client-dropdown" className="block text-s font-bold text-gray-600 dark:text-gray-300">
              Client
            </label>
            <p className="text-gray-600 dark:text-gray-300 text-sm">Select the client you want to fetch monitored titles from.</p>
            <select
              id="client-dropdown"
              value={selectedClient}
              onChange={(e) => setSelectedClient(e.target.value)}
              className="block w-1/2 mt-2 focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
              disabled={isFetchingClients}
            >
              {clients?.map((client) => (
                <option key={client.id} value={client.name}>{client.name}</option>
              ))}
            </select>
          </div>
        )}
        {sourceType === "lists" && (
          <div className="mb-4">
            <label htmlFor="lists-dropdown" className="block text-s font-bold text-gray-600 dark:text-gray-300">
              Lists
            </label>
            <p className="text-gray-600 dark:text-gray-300 text-sm">Select the list you want to fetch titles from.</p>
            <select
              id="lists-dropdown"
              value={selectedList}
              onChange={(e) => setSelectedList(e.target.value)}
              className="block w-1/2 mt-2 focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
            >
              {listOptions.map((list) => (
                <option key={list.url} value={list.url}>{list.name}</option>
              ))}
            </select>
          </div>
        )}
  
        {/* Filters dropdown - always visible */}
        <div className="mb-4">
          <label htmlFor="filter-dropdown" className="block text-s font-bold text-gray-600 dark:text-gray-300">
            Filter
          </label>
          <p className="text-gray-600 dark:text-gray-300 text-sm">Select the filter you want to populate the titles with.</p>
          <select
            id="filter-dropdown"
            value={selectedFilter}
            onChange={(e) => setSelectedFilter(e.target.value)}
            className="block w-1/2 mt-2 focus:outline-none focus:ring-1 focus:ring-offset-0 focus:ring-blue-500 dark:focus:ring-blue-500 rounded-md sm:text-sm border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
            disabled={isFetchingFilters}
          >
            {filters?.map((filter) => (
              <option key={filter.id} value={filter.name}>{filter.name}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-box items-">
          <button
            type="button"
            className="relative inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-bold rounded-md text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
          >
            <PlusIcon className="h-5 w-5 mr-1" />
          Automate
          </button>
          <p className="pl-4 text-gray-600 dark:text-gray-300 text-sm py-2">The filter will be updated every X hours.</p>
        </div>
      </div>
      <div className="py-6 px-4 sm:p-6">
        {/* ... (automate the filter button and output display) */}
      </div>
    </div>
  );
  
};

export default Omegabrr;
