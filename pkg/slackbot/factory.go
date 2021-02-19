package slackbot

import (
	slackapp "github.com/jenkins-x-plugins/slack/pkg/apis/slack/v1alpha1"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"k8s.io/apimachinery/pkg/util/runtime"

	"k8s.io/client-go/kubernetes"

	jenkinsv1client "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
)

type SlackOptions struct {
	Dir        string `env:"GIT_DIR"`
	SlackToken string `env:"SLACK_TOKEN"`
	SlackURL   string `env:"SLACK_URL"`
	GitURL     string `env:"GIT_URL"`
	Name       string
	Namespace  string
}

// SlackBotOptions contains options for the SlackBot
type SlackBotOptions struct {
	SlackOptions
	KubeClient        kubernetes.Interface
	JXClient          jenkinsv1client.Interface
	SlackClient       *slack.Client
	ScmClient         *scm.Client
	SourceConfigs     *v1alpha1.SourceConfig
	Statuses          slackapp.Statuses
	Timestamps        map[string]map[string]*MessageReference
	SlackUserResolver SlackUserResolver
	GitClient         gitclient.Interface
	CommandRunner     cmdrunner.CommandRunner
}

// Validate configures the clients for the slack bot
func (o *SlackBotOptions) Validate() error {
	if o.SlackClient == nil {
		if o.SlackToken == "" {
			return errors.Errorf("no $SLACK_TOKEN defined")
		}
		if o.SlackURL != "" {
			log.Logger().Infof("using slack URL %s", o.SlackURL)
			o.SlackClient = slack.New(o.SlackToken, slack.OptionAPIURL(o.SlackURL))
		} else {
			o.SlackClient = slack.New(o.SlackToken)
		}
	}

	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return err
	}

	o.JXClient, err = jxclient.LazyCreateJXClient(o.JXClient)
	if err != nil {
		return err
	}

	if o.ScmClient == nil {
		o.ScmClient, err = factory.NewClientFromEnvironment()
		if err != nil {
			return errors.Wrapf(err, "failed to create SCM client")
		}
	}
	o.SlackUserResolver = NewSlackUserResolver(o.SlackClient, o.JXClient, o.Namespace)

	if o.Dir == "" {
		if o.GitURL == "" {
			return errors.Errorf("no $GIT_URL defined")
		}
		if o.GitClient == nil {
			o.GitClient = cli.NewCLIClient("", o.CommandRunner)
		}
		o.Dir, err = gitclient.CloneToDir(o.GitClient, o.GitURL, "")
		if err != nil {
			return errors.Wrapf(err, "failed to clone git URL %s", o.GitURL)
		}
	}
	o.SourceConfigs, err = sourceconfigs.LoadSourceConfig(o.Dir, true)
	if err != nil {
		return errors.Wrapf(err, "failed to load source configs from dir %s", o.Dir)
	}

	return nil
}

func (o *SlackBotOptions) Run() error {
	defer runtime.HandleCrash()

	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	log.Logger().Infof("Watching slackbots in namespace %s\n", o.Namespace)

	o.WatchActivities()
	return nil
}

func (o *SlackBotOptions) previousPipelineFailed(activity *jenkinsv1.PipelineActivity) (bool, error) {
	// TODO...
	return false, nil

}
