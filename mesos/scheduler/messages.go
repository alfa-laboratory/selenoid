package scheduler

//Универсальная структура для хранения  ID
type ID struct {
	Value string `json:"value"`
}

type Docker struct {
	Image        string          `json:"image"`
	Network      string          `json:"network"`
	Privileged   bool            `json:"privileged"`
	PortMappings *[]PortMappings `json:"port_mappings"`
}

//Структура для хранения данных о контейнере
type Container struct {
	Type   string `json:"type"`
	Docker Docker `json:"docker"`
}

type PortMappings struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
	Name          string `json:"name"`
}

//Резервируемые ресурсы
type Resource struct {
	Name   string  `json:"name"`
	Ranges *Ranges `json:"ranges,omitempty"`
	Role   string  `json:"role,omitempty"`
	Type   string  `json:"type"`
	Scalar *Scalar `json:"scalar,omitempty"`
}

type Scalar struct {
	Value float64 `json:"value,numbers"`
}

type Ranges struct {
	Range [] Range `json:"range"`
}

type Range struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

//Структура для хранения таски запуска
type Launch struct {
	TaskInfos []TaskInfo `json:"task_infos"`
}

type Command struct {
	Shell bool `json:"shell"`
}

type TaskInfo struct {
	Name      string     `json:"name"`
	TaskID    ID         `json:"task_id"`
	AgentID   ID         `json:"agent_id"`
	Command   Command    `json:"command"`
	Container *Container `json:"container"`
	Resources []Resource `json:"resources"`
}

type FrameworkInfo struct {
	User  string   `json:"user"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

type Subscribe struct {
	FrameworkInfo FrameworkInfo `json:"framework_info"`
}

type SubscribeMessage struct {
	Type      string    `json:"type"`
	Subscribe Subscribe `json:"subscribe"`
}

type Decline struct {
	OfferIds []ID    `json:"offer_ids"`
	Filters  Filters `json:"filters"`
}

type DeclineMessage struct {
	FrameworkID ID      `json:"framework_id"`
	Type        string  `json:"type"`
	Decline     Decline `json:"decline"`
}

type Filters struct {
	RefuseSeconds float64 `json:"refuse_seconds"`
}

type Accept struct {
	OfferIds   []ID         `json:"offer_ids"`
	Operations *[]Operation `json:"operations"`
	Filters    Filters      `json:"filters"`
}

type AcceptMessage struct {
	FrameworkID ID     `json:"framework_id"`
	Type        string `json:"type"`
	Accept      Accept `json:"accept"`
}

type Operation struct {
	Type   string  `json:"type"`
	Launch *Launch `json:"launch"`
}

type AcknowledgeMessage struct {
	FrameworkID ID          `json:"framework_id"`
	Type        string      `json:"type"`
	Acknowledge Acknowledge `json:"acknowledge"`
}

type Acknowledge struct {
	AgentID ID     `json:"agent_id"`
	TaskID  ID     `json:"task_id"`
	UUID    string `json:"uuid"`
}

type Kill struct {
	TaskID ID `json:"task_id"`
}

type KillMessage struct {
	FrameworkID ID     `json:"framework_id"`
	Type        string `json:"type"`
	Kill        Kill   `json:"kill"`
}

func newPortMappings(portRange Range, enableVNC bool) *[]PortMappings {
	portMappings := []PortMappings{newMapping(4444, portRange.Begin)}
	if enableVNC {
		portMappings = append(portMappings, newMapping(5900, portRange.End))
	}
	return &portMappings
}

func newMapping(containerPort int, hostPort int) PortMappings {
	return PortMappings{
		ContainerPort: containerPort,
		Name:          "http",
		HostPort:      hostPort,
		Protocol:      "tcp"}
}

func newContainer(portRange Range, task Task) *Container {
	return &Container{
		Type: "DOCKER",
		Docker: Docker{
			Image:        task.Image,
			Network:      "BRIDGE",
			Privileged:   true,
			PortMappings: newPortMappings(portRange, task.EnableVNC),
		},
	}
}

func newResourcePorts(portRange Range) Resource {
	var rangePort = Range{
		Begin: portRange.Begin,
		End:   portRange.End,
	}

	return Resource{
		Type: "RANGES",
		Name: "ports",
		Ranges: &Ranges{
			[]Range{rangePort},
		},
		Role: "*",
	}
}

func newResourcesContainer(name string, value float64) Resource {
	return Resource{
		Type:   "SCALAR",
		Name:   name,
		Scalar: &Scalar{value},
	}
}

func newLaunchTaskInfo(resource ResourcesForOneTask, task Task) *Launch {

	var taskInfo = TaskInfo{
		Name:      "My Task",
		TaskID:    ID{task.TaskId},
		AgentID:   resource.AgentId,
		Command:   Command{false},
		Container: newContainer(resource.Range, task),
		Resources: []Resource{
			newResourcePorts(resource.Range),
			newResourcesContainer("cpus", CpuLimit),
			newResourcesContainer("mem", MemLimit),
		},
	}

	return &Launch{TaskInfos: []TaskInfo{taskInfo}}
}

func newOperations(resources []ResourcesForOneTask, tasks []Task) *[]Operation {
	var operations []Operation
	for i, task := range tasks {
		operations = append(operations, Operation{
			Type:   "LAUNCH",
			Launch: newLaunchTaskInfo(resources[i], task),
		})
	}
	return &operations
}

func (scheduler *Scheduler) newAcceptMessage(resources []ResourcesForOneTask, tasks []Task) (AcceptMessage) {
	return AcceptMessage{
		FrameworkID: scheduler.FrameworkId,
		Type:        "ACCEPT",
		Accept: Accept{
			getUniqueOfferIds(resources),
			newOperations(resources, tasks),
			Filters{RefuseSeconds: 1.0},
		},
	}
}

func getUniqueOfferIds(resources []ResourcesForOneTask) []ID {
	offersMap := make(map[ID]bool)
	var set []ID
	for _, v := range resources {
		if !offersMap[v.OfferId] {
			offersMap[v.OfferId] = true
		}
	}
	for k,_ := range offersMap {
		set = append(set, k)
	}
	return set
}


func newSubscribedMessage(user string, name string, roles []string) (SubscribeMessage) {
	return SubscribeMessage{
		Type: "SUBSCRIBE",
		Subscribe: Subscribe{
			FrameworkInfo{
				User:  user,
				Name:  name,
				Roles: roles,
			},
		},
	}
}

func newAcknowledgeMessage(frameworkId ID, agentId ID, UUID string, taskId ID) (AcknowledgeMessage) {
	return AcknowledgeMessage{
		FrameworkID: frameworkId,
		Type:        "ACKNOWLEDGE",
		Acknowledge: Acknowledge{
			AgentID: agentId,
			TaskID:  taskId,
			UUID:    UUID,
		},
	}
}

func newDeclineMessage(frameworkId ID, offerId []ID) (DeclineMessage) {
	return DeclineMessage{
		FrameworkID: frameworkId,
		Type:        "DECLINE",
		Decline: Decline{
			OfferIds: offerId,
			Filters: Filters{
				RefuseSeconds: 1.0,
			},
		},
	}
}

func newKillMessage(frameworkId ID, taskId string) (KillMessage) {
	return KillMessage{
		FrameworkID: frameworkId,
		Type:        "KILL",
		Kill: Kill{
			TaskID: ID{taskId},
		},
	}
}
