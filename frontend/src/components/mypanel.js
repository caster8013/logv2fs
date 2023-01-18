import { useEffect, useState } from "react";
import { alert, reset } from "../store/message";
import { useSelector, useDispatch } from "react-redux";
import { formatBytes } from "../service/service";
import axios from "axios";
import TapToCopied from "./tapToCopied";
import TrafficTable from "./trafficTable";
import Alert from "./alert";

function Mypanel() {
	const [user, setUser] = useState({});

	const dispatch = useDispatch();
	const loginState = useSelector((state) => state.login);
	const message = useSelector((state) => state.message);
	const rerenderSignal = useSelector((state) => state.rerender);

	useEffect(() => {
		if (message.show === true) {
			setTimeout(() => {
				dispatch(reset({}));
			}, 5000);
		}
	}, [message, dispatch]);

	useEffect(() => {
		axios
			.get(process.env.REACT_APP_API_HOST + "user/" + loginState.jwt.Email, {
				headers: { token: loginState.token },
			})
			.then((response) => {
				setUser(response.data);
			})
			.catch((err) => {
				dispatch(alert({ show: true, content: err.toString() }));
			});
	}, [loginState, dispatch, rerenderSignal]);

	return (
		<div className="py-3 flex-1">
			<Alert message={message.content} type={message.type} shown={message.show} close={() => { dispatch(reset({})); }} />
			<div className="flex flex-col md:flex-row">
				<div className="grow p-6 m-3 bg-white rounded-lg border border-gray-200 shadow-md hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:hover:bg-gray-700">
					<div className="h3">
						{user.used_by_current_day &&
							formatBytes(user.used_by_current_day.amount)}
					</div>
					<p>
						Traffic Used Today (
						{user.used_by_current_day && user.used_by_current_day.period})
					</p>
				</div>
				<div className="grow p-6 m-3 md:mx-2 bg-white rounded-lg border border-gray-200 shadow-md hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:hover:bg-gray-700">
					<div className="h3">
						{user.used_by_current_month &&
							formatBytes(user.used_by_current_month.amount)}
					</div>
					<p>
						Traffic Used This Month (
						{user.used_by_current_month && user.used_by_current_month.period})
					</p>
				</div>
				<div className="grow p-6 m-3 bg-white rounded-lg border border-gray-200 shadow-md hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:hover:bg-gray-700">
					<div className="h3">{user && formatBytes(user.used)}</div>
					<p>Traffic Used In Total</p>
				</div>
			</div>
			<div
				// className="w-full flex justify-center"
				className="w-full flex justify-center bg-white border border-gray-200 rounded-lg shadow-md sm:p-6 md:p-8 dark:bg-gray-800 dark:border-gray-700"
			>
				<form
					// className="space-y-6" 
					action="#">
					<h5 className="text-xl font-medium text-gray-900 dark:text-white">My Info</h5>
					<div>
						<label for="email" className="block mb-2 text-sm font-medium text-gray-900 dark:text-white">用户名: </label>
						<TapToCopied>{user.email}</TapToCopied>
						{/* <input type="email" name="email" id="email" className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-600 dark:border-gray-500 dark:placeholder-gray-400 dark:text-white" placeholder="name@company.com" required /> */}
					</div>
					<div>
						<label for="password" className="block mb-2 text-sm font-medium text-gray-900 dark:text-white">path:</label>
						<TapToCopied>{user.path}</TapToCopied>
						{/* <input type="password" name="password" id="password" placeholder="••••••••" className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-600 dark:border-gray-500 dark:placeholder-gray-400 dark:text-white" required /> */}
					</div>
					<div>
						<label for="password" className="block mb-2 text-sm font-medium text-gray-900 dark:text-white">uuid:</label>
						<TapToCopied>{user.uuid}</TapToCopied>
						{/* <input type="password" name="password" id="password" placeholder="••••••••" className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-600 dark:border-gray-500 dark:placeholder-gray-400 dark:text-white" required /> */}
					</div>
					<div>
						<label for="password" className="block mb-2 text-sm font-medium text-gray-900 dark:text-white">SubUrl:</label>
						<TapToCopied>
							{process.env.REACT_APP_FILE_AND_SUB_URL + "/static/" + user.email}
						</TapToCopied>
						{/* <input type="password" name="password" id="password" placeholder="••••••••" className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-600 dark:border-gray-500 dark:placeholder-gray-400 dark:text-white" required /> */}
					</div>
					<div>
						<label for="password" className="block mb-2 text-sm font-medium text-gray-900 dark:text-white">Clash YAML:</label>
						<TapToCopied>
							{process.env.REACT_APP_FILE_AND_SUB_URL + "/clash/" + user.email + ".yaml"}
						</TapToCopied>
						{/* <input type="password" name="password" id="password" placeholder="••••••••" className="bg-gray-50 border border-gray-300 text-gray-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 dark:bg-gray-600 dark:border-gray-500 dark:placeholder-gray-400 dark:text-white" required /> */}
					</div>
				</form>
			</div>

			{
				user.traffic_by_day && user.traffic_by_day.length > 0 && (

					<div className="">
						<div className="px-3 flex flex-col">
							<div className="text-4xl my-3 text-center">
								Monthly Traffic in the Past 1 Year
							</div>
							<TrafficTable data={user.traffic_by_month} limit={12} by="月份" />
						</div>
						<div className="flex flex-col">
							<div className="text-4xl my-3 text-center">
								Daily Traffic in the Past 3 Month
							</div>
							<TrafficTable data={user.traffic_by_day} limit={90} by="日期" />
						</div>
					</div>
				)
			}

		</div>
	);
}

export default Mypanel;
