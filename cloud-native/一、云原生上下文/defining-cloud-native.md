什么是云原生
-----------

云原生软件是高度分布式的，必须在一个不断变化的环境中运行，而且自身也在不断地发生变化。

![](../assets/figure_1-5.png)

* 如果想让软件**始终处于运行**(Always up)状态，则必须对基础设施的故障和需求变更具有更好的**弹性**（Resilience）。无论这些故障和变更是计划内的还是计划外的。没有一个实体可以保证永远不会失败，所以会在整个设计中都考了**冗余**（Redundancy）的情况。
* 由更小、更松耦合、可独立部署和发布的组件（通常称为微服务）组成的软件，可以支持更**敏捷**（Agility）的发布模型。
* 大量的设备会导致请求和数据量大幅波动，因此需要能够**动态伸缩**（Dynamic scalability）以及持续提供服务的软件。
* 将软件拆分成一组独立的组件，冗余地部署多个实例，这就意味着**分布式**（Distributed）。
* 频繁地发布意味着频繁的修改，通过水平伸缩的运维方式来适应波动的请求，也意味着需要不断调整（Constantly changing）。

> 云原生软件架构通常被称为**微服务**，但是云原生架构包含与云原生元应程序重要的两项特性：**数据和交互**。
>
> “云”是指我们在哪里计算（云服务商的数据中心，AWS、Azure或者Aliyun等），而“云原生”指的是如何实现。

**失败和变化时正常规律，而不是例外。混沌测试可以帮助我们为错误和失败做好准备，通过不断模拟各种出错情况，能够及早发现任何系统性缺陷并及时修复。**



### 现代应用程序的需求

我们希望应用程序总是可用的，能够不断地升级并提供个性化的体验。用户的需求可以概括为：**持续的可用性、频繁发布新版本以实现不断演进、易于伸缩和智能化。**

#### 零停机时间

现代的应用程序的一个关键需求：**它必须是始终可用的**。容忍应用程序出现短暂不可用情况的日子已经一去不复返了。

维护系统的正常运行不仅是运维团队的问题，更是软件开发者或者架构师设计和开发一个松耦合、组件化的系统的责任。通过部署冗余组件来应对不可避免的故障，并设置隔离机制来防止故障在整个系统中引起连锁反应。还必须把软件设计成能够在不停机的情况下完成计划事件（如，升级）。

#### 缩短反馈周期

上线一个新功能需要承担一定程度的风险，最佳的办法是发布一个功能的早期版本并收集反馈信息。这就需要在**应用程序具有频繁发布代码的能力**，这不仅有助于让用户感到兴奋，同时**有助于降低风险和缩短用户的反馈周期**。

#### 移动端和多设备支持

用户越来越希望他们的应用体验能随时从一个设备无缝切换到另一个设备上。这就要求**核心服务必须能够支持所有为用户提供服务的终端设备**，并且系统必须能够适应不断变化的需求。

#### 互联设备（物联网）

物联网设备从两个基本方面改变了应用程序的特性。首先是**急剧增加的互联网流量**，其次是**收集和处理这些海量数据**，用于计算的基础设施势必发生重大的改变。

#### 数据驱动

**由于数据量正在不断增加，数据源分布地更加广泛，而软件的交付周期正在缩短**。这就使得大型、集中式、共享的数据库变得无法使用。应用程序需要的是一个由更小的、本地化数据库组成的网络，以及能够在多个数据库管理系统之间管理关系的软件。



### 云原生软件的思维模式

#### 基本的软件架构中的常见元素

* 云原生应用程序：由编写的大量业务代码构成，实现软件的业务逻辑。
* 云原生数据：这是在云原生软件中存储状态的地方。云原生软件将代码分解成许多更小的模块（应用程序），同样数据库也被拆分成多个且分布式的。
* 云原生交互：云原生软件是云原生应用程序和云原生数据的组合，这些实体（服务）之间的交互方式最终决定了数字化解决方案的功能和质量。

#### 云原生应用程序的问题

* 应用程序可以通过添加或删除实例来进行容量伸缩。
* 当某一个实例出现故障时，如何将故障实例与整个应用程序集群隔离（更容易地执行恢复操作）。
* 当应用程序部署了许多实例，并且它们所运行的环境不断变化（不在用一台机器上，甚至容器内），云原生应用程序如何加载各自的配置。
* 基于云环境的动态特性要求我们改变管理应用程序生命周期的方式（如何在新的上下文中启动、配置、重新配置和关闭应用程序）。

#### 云原生数据的问题

* 需要打破数据统一管理的观念。大型、单一的数据库不适用于现代化的软件，它发展缓慢而且脆弱，缺少松耦合架构的敏捷性和健壮性。
* 分布式的数据架构由一些独立、适用于特定场景的数据库，以及一些数据源自其它地方、仅用于数据分析的数据库组成。
* 当多个数据库中存在多个实体，必须解决公共信息在多个不同实例之间的同步问题。
* 分布式数据架构的核心就是将状态作为一系列事件的结果来处理。

#### 云原生交互的问题

* 当一个应用程序有多个实例时们需要使用某种路由系统来选择性地访问实例。这就会使用到同步的请求/响应模式或者异步的事件驱动模式。
* 在高度分布式、不断变化的环境中，必须考虑访问失败的情况。自动重试是云原生软件中的一种常用模式，同时在应用自动重试模式时，断路器必不可少。
* 指标监控、日志服务也必须针对新的服务架构加以调整。
* 这些独立的部分最终会组合成一个更大的整体，所以它们之间的底层交互协议必须适合云原生环境。



### 应用云原生

当拥抱新的架构模式和运维实践时，所开发的软件就会非常适合于在云环境中工作，如同天生源自于此。

#### 什么情况下不需要应用云原生

* 软件和计算基础设施不需要云计算。例如软件不是分布式的，并且很少出现变化，那么完全不同做到像大规模运行的Web或移动应用一样的稳定性。再比如运行于洗衣机或者电饭煲中的代码。
* 云原生软件的特点并不适合解决所面临的问题。如果系统需要**强一致性**时，就不能使用这些新模式了。**最终一致性**是许多云原生模式的核心。
* 虽然现有的软件不是云原生的，重写成云原生的也没有什么价值。

#### 云原生的价值

云原生的绝妙之处在于它最终是由需要不同的组件组成的，即使一些组件的模式不是最新的，云原生的组件了仍然可以与它们进行交互。**值得注意的是，在重构遗留代码时，不需要一下子完成所有重构工作。在向云迁移的过程中，即使只有一部分解决方案是云原生的，也是有价值的。**

