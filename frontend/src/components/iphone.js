import { useSelector } from "react-redux";
import TapToCopied from "./tapToCopied";

function Ihpone() {
	const loginState = useSelector((state) => state.login);

	return (
		<div className="md:container md:mx-auto px-20">
			<h1 className="my-4 px-auto text-4xl font-extrabold tracking-tight leading-none text-gray-900 md:text-5xl lg:text-6xl dark:text-white">
			iphone/ipad 中安装 Shadowrocket
			</h1>
			<p className="my-6 text-baseg font-normal text-gray-500 lg:text-xl sm:px-16 xl:px-10 dark:text-gray-400">Step 1: 安装 Shadowrocket 客户端</p>
			<div className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-10 dark:text-sky-200">
				<p>依次点按 settings -{'>'} Apple ID、iCloud、媒体与购买项目 -{'>'} 媒体与购买项目</p>
				<p>退出你的Apple ID, 用下面的 Apple ID(美区)登入</p>
			</div>
			<p className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-10 dark:text-sky-200">
				<code>
				Apple ID: <TapToCopied>warley8013@gmail.com</TapToCopied><br />
				Password: <TapToCopied>Google#2006</TapToCopied><br />
				</code>
				<code className="text-sm py-5">
					Notice: <br/>
					1. 上面apple id已经购买shadowrocket, 登陆后可直接安装。<br />
					2. apple ID 登陆时, 需要双重认证, 给我发信息, 我会发你认证数字。<br />
				</code>
			</p>
			<div className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-10 dark:text-sky-200">
				<p>打开app store, 查找 "shadowrocket" , 安装</p>
			</div>
			<p className="my-6 text-baseg font-normal text-gray-500 lg:text-xl sm:px-16 xl:px-10 dark:text-gray-400">Step 2: 添加配置</p>
			<div className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-12 dark:text-sky-200">
				<p>打开 Shadowrocket, 点按shadowrocket右上角“+”号</p>
				<p>类型选择“Subscribe”,URL 填写下面的地址</p>
				<TapToCopied>{process.env.REACT_APP_FILE_AND_SUB_URL + "/static/" + loginState.jwt.Email} </TapToCopied>
				<p>备注填"clash"</p>
				<p>然后点按右上角的“完成”按钮。配置添加成功的话, 回到 Shadowrocket 首页, 有新的项目生成！</p>
				<p></p>
			</div>
			<p className="my-6 text-baseg font-normal text-gray-500 lg:text-xl sm:px-16 xl:px-10 dark:text-gray-400">Step 3: 日常运行设置</p>
			<div className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-10 dark:text-sky-200">
				<p>回到 Shadowrocket 首页</p>
				<p>全局路由选择，配置</p>
				<p>点按新生成项目左边箭头，选中一个服务器</p>
				<p>点按"未连接"右边单选框，授权 Shadowrocket 接入网络</p>
			</div>
			<div className="my-6 text-baseg font-normal text-gray-500 lg:text-base sm:px-16 xl:px-10 dark:text-sky-200">
				打开浏览器，访问 <TapToCopied>https://www.google.com</TapToCopied>，如果能正常访问，说明配置成功。
			</div>
		</div>
	);
}

export default Ihpone;
