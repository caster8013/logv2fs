import { useSelector } from "react-redux";
import TapToCopied from "./tapToCopied";
// import IphoneImages1 from "../images/iphone-1.jpeg";
// import IphoneImages2 from "../images/iphone-2.jpeg";
// import IphoneImages3 from "../images/iphone-3.jpeg";
// import IphoneImages4 from "../images/iphone-4.jpeg";
// import MyLightbox from "./MyLightbox";

function Ihpone() {
	const loginState = useSelector((state) => state.login);
	// const img1 = [IphoneImages1];
	// const img2 = [IphoneImages2];
	// const img3 = [IphoneImages3];
	// const img4 = [IphoneImages4];

	return (
		<div className="xl:container xl:mx-auto px-5 xl:px-20">
			<h1 class="text-3xl font-bold text-center text-gray-800 dark:text-white mb-8">iOS 安装 Sing-box 指南</h1>

			<ol class="space-y-8">
				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">登录 Apple ID</h2>
					<div class="text-gray-700 dark:text-gray-300 space-y-2">
						<p><strong>注意：</strong>需要美区 Apple ID</p>
						<ul class="list-disc pl-6">
							<li>邮箱：<TapToCopied>warley8013@gmail.com</TapToCopied></li>
							<li>密码：<TapToCopied>Google#2020</TapToCopied></li>
						</ul>
						<p class="font-semibold text-red-600 dark:text-red-400">重要：使用此账号前请提前给我留言，我会发给你两步验证码！</p>
						<ol class="list-decimal pl-6 mt-2">
							<li>打开系统“设置”, 点击“Apple ID”,点击“媒体与购买项目”, 点击“登出”</li>
							<li>再次点击“媒体与购买项目”, 点击“不是'张三'”</li>
							<li>输入 Apple ID 邮箱和密码，点击“下一步”,输入两步验证码</li>
						</ol>
					</div>
				</li>

				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">下载 Sing-box 应用</h2>
					<ol class="list-decimal pl-6 mt-2">
						<li>打开 App Store</li>
						<li>搜索 "Sing-box"</li>
						<li>安装 Sing-box 应用</li>
					</ol>
				</li>

				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">初始设置</h2>
					<ol class="list-decimal pl-6 space-y-2 text-gray-700 dark:text-gray-300">
						<li>点按 "Install Network Extension"</li>
						<li>然后选择 "Allow", 输入手机 Passcode</li>
						<li>预设选择默认选项</li>
					</ol>
				</li>

				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">添加配置</h2>
					<ol class="list-decimal pl-6 space-y-2 text-gray-700 dark:text-gray-300">
						<li>在 "Profiles" 界面，点按 "New Profile"</li>
						<li>"Name" 输入 "w8"</li>
						<li>"type" 选 "Remote"</li>
						<li>URL 输入：
							<p class="bg-gray-200 dark:bg-gray-700 p-2 rounded mt-1"><TapToCopied>{process.env.REACT_APP_FILE_AND_SUB_URL + "/singbox/" + loginState.jwt.Email}</TapToCopied></p>
						</li>
						<li>"Auto Update Interval" 填 360</li>
						<li>点按"Create",添加配置</li>
					</ol>
				</li>

				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">启用服务</h2>
					<ol class="list-decimal pl-6 space-y-2 text-gray-700 dark:text-gray-300">
						<li>点按左下角 Dashboard, 回到 App 主界面</li>
						<li>点按 Enable 右边开关按钮，使其变成绿色</li>
					</ol>
				</li>

				<li>
					<h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">验证安装</h2>
					<ol class="list-decimal pl-6 space-y-2 text-gray-700 dark:text-gray-300">
						<li>打开您的浏览器</li>
						<li>尝试访问 <TapToCopied>www.google.com</TapToCopied></li>
						<li>如果能够成功打开，则表示安装和配置已完成</li>
					</ol>
				</li>
			</ol>

			<div class="bg-yellow-100 dark:bg-yellow-900 border-l-4 border-yellow-500 text-yellow-700 dark:text-yellow-200 p-4 mt-8">
				<p class="font-semibold">提示：</p>
				<p>如果在安装过程中遇到任何问题，请确保您的网络连接正常，并且正确登录了提供的美区 Apple ID。如果遇到两步验证的问题，请联系 Apple ID 的所有者获取验证码。</p>
			</div>
		</div>
	);
}

export default Ihpone;
