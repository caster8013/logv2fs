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
					<h4 className="text-3xl font-extrabold dark:text-white">
						{ user?.daily_logs?.length > 0 ? formatBytes(user?.daily_logs?.slice(-1)[0].traffic) : "0 Bytes" }
					</h4>
					<p>
						Used Today
					</p>
				</div>
				<div className="grow p-6 m-3 md:mx-2 bg-white rounded-lg border border-gray-200 shadow-md hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:hover:bg-gray-700">
					<h4 className="text-3xl font-extrabold dark:text-white">
						{ user?.monthly_logs?.length > 0 ? formatBytes(user?.monthly_logs?.slice(-1)[0].traffic) : "0 Bytes" }
					</h4>
					<p>
						Used This Month
					</p>
				</div>
				<div className="grow p-6 m-3 bg-white rounded-lg border border-gray-200 shadow-md hover:bg-gray-100 dark:bg-gray-800 dark:border-gray-700 dark:hover:bg-gray-700">
					<h4 className="text-3xl font-extrabold dark:text-white">
						{ user?.yearly_logs?.length > 0 ? formatBytes(user?.yearly_logs?.slice(-1)[0].traffic) : "0 Bytes" }
					</h4>
					<p>Used This Year</p>
				</div>
			</div>
			<div className="w-full md:w-3/4 mx-auto flex justify-center bg-white border border-gray-200 rounded-lg shadow-md sm:p-6 md:p-8 dark:bg-gray-800 dark:border-gray-700" >
				<div>
					<h5 className="text-4xl py-2 font-extrabold text-gray-900 dark:text-white">Basic Info</h5>
					<div className="py-1 flex justify-between items-center">
						<pre className="inline text-sm font-medium text-gray-900 dark:text-white">Email: </pre>
						<TapToCopied>{user.email_as_id}</TapToCopied>
					</div>
					<div className="py-1 flex justify-between items-center">
						<pre className="inline  text-sm font-medium text-gray-900 dark:text-white">SubUrl:</pre>
						<TapToCopied>
							{process.env.REACT_APP_FILE_AND_SUB_URL + "/static/" + user.email_as_id}
						</TapToCopied>
					</div>
					<div className="py-1 flex justify-between items-center">
                        <pre className="inline  text-sm font-medium text-gray-900 dark:text-white">Verge:</pre>
                        <TapToCopied>
                            {process.env.REACT_APP_FILE_AND_SUB_URL + "/verge/" + user.email_as_id }
                        </TapToCopied>
                    </div>
                    <div className="py-1 flex justify-between items-center">
                        <pre className="inline  text-sm font-medium text-gray-900 dark:text-white">Sing-box:</pre>
                        <TapToCopied>
                            {process.env.REACT_APP_FILE_AND_SUB_URL + "/singbox/" + user.email_as_id}
                        </TapToCopied>
                    </div>
				</div>
			</div>

			{
				user?.daily_logs?.length > 0 && (

					<div className="">
						<div className="px-3 flex flex-col">
							<div className="text-4xl my-3 text-center">
								Monthly Used in the Past 1 Year
							</div>
							<TrafficTable data={user?.monthly_logs} limit={12} by="月份" />
						</div>
						<div className="flex flex-col">
							<div className="text-4xl my-3 text-center">
								Daily Used in the Past 3 Months
							</div>
							<TrafficTable data={user?.daily_logs} limit={90} by="日期" />
						</div>
					</div>
				)
			}

		</div>
	);
}

export default Mypanel;
