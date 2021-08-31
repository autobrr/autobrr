import { FC } from 'react'
import { XIcon, CheckCircleIcon, ExclamationIcon, ExclamationCircleIcon } from '@heroicons/react/solid'
import { toast } from 'react-hot-toast'

type Props = {
  type: 'error' | 'success' | 'warning'
  body?: string
  t?: any;
}

const Toast: FC<Props> = ({
  type,
  body,
  t
}) => {
  return (
    <div className={`${
      t.visible ? 'animate-enter' : 'animate-leave'
    } max-w-sm w-full bg-white shadow-lg rounded-lg pointer-events-auto ring-1 ring-black ring-opacity-5 overflow-hidden transition-all`}>
      <div className="p-4">
        <div className="flex items-start">
          <div className="flex-shrink-0">
            {type === 'success' && <CheckCircleIcon className="h-6 w-6 text-green-400" aria-hidden="true" />}
            {type === 'error' && <ExclamationCircleIcon className="h-6 w-6 text-red-400" aria-hidden="true" />}
            {type === 'warning' && <ExclamationIcon className="h-6 w-6 text-yellow-400" aria-hidden="true" />}
          </div>
          <div className="ml-3 w-0 flex-1 pt-0.5">
            <p className="text-sm font-medium text-gray-900">
              {type === 'success' && "Success"}
              {type === 'error' && "Error"}
              {type === 'warning' && "Warning"}
            </p>
            <p className="mt-1 text-sm text-gray-500">{body}</p>
          </div>
          <div className="ml-4 flex-shrink-0 flex">
            <button
              className="bg-white rounded-md inline-flex text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              onClick={() => {
                toast.dismiss(t.id)
              }}
            >
              <span className="sr-only">Close</span>
              <XIcon className="h-5 w-5" aria-hidden="true" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Toast;