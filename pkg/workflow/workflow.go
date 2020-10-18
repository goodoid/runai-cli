package workflow

import (
	"fmt"
	"os"
	"strconv"

	"io/ioutil"

	"github.com/run-ai/runai-cli/pkg/util/helm"
	"github.com/run-ai/runai-cli/pkg/util/kubectl"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	FamilyNameLabelSelectorName = "FamilyName"
	FamilyIndexLabelSelectorName = "FamilyIndex"
	ConfigMapGenerationRetries = 5
	)

type JobFiles struct {
	valueFileName   string
	envValuesFile   string
	template        string
	appInfoFileName string
}

/**
*	delete training job with the job name
**/

func DeleteJob(namespace, jobName string) error {
	appInfoFileName, err := kubectl.SaveAppConfigMapToFile(jobName, "app", namespace)
	if err != nil {
		log.Debugf("Failed to SaveAppConfigMapToFile due to %v", err)
	} else {
		result, err := kubectl.UninstallAppsWithAppInfoFile(appInfoFileName, namespace)
		log.Debugf("%s", result)
		if err != nil {
			log.Warnf("Failed to remove some of the job's resources, they might have been removed manually and not by using Run:AI CLI.")
		}
	}

	err = kubectl.DeleteAppConfigMap(jobName, namespace)
	if err != nil {
		log.Warningf("Delete configmap %s failed, please clean it manually due to %v.", jobName, err)
		log.Warningf("Please run `kubectl delete -n %s cm %s`", namespace, jobName)
		return err
	}

	return nil
}

/**
*	Submit training job
**/

func getDefaultValuesFile(environmentValues string) (string, error) {
	valueFile, err := ioutil.TempFile(os.TempDir(), "values")
	if err != nil {
		return "", err
	}

	_, err = valueFile.WriteString(environmentValues)

	if err != nil {
		return "", err
	}

	log.Debugf("Wrote default cluster values file to path %s", valueFile.Name())

	return valueFile.Name(), nil
}

func generateJobFiles(name string, namespace string, values interface{}, environmentValues string, chart string) (*JobFiles, error) {
	valueFileName, err := helm.GenerateValueFile(values)
	if err != nil {
		return nil, err
	}

	envValuesFile := ""
	if environmentValues != "" {
		envValuesFile, err = getDefaultValuesFile(environmentValues)
		if err != nil {
			log.Debugln(err)
			return nil, fmt.Errorf("Error getting default values file of cluster")
		}
	}

	if err != nil {
		log.Debugln(err)
		return nil, fmt.Errorf("Error getting default values file of cluster")
	}

	// 2. Generate Template file
	template, err := helm.GenerateHelmTemplate(name, namespace, valueFileName, envValuesFile, chart)
	if err != nil {
		return nil, err
	}

	// 3. Generate AppInfo file
	appInfoFileName, err := kubectl.SaveAppInfo(template, namespace)
	if err != nil {
		return nil, err
	}

	jobFiles := &JobFiles{
		valueFileName:   valueFileName,
		envValuesFile:   envValuesFile,
		template:        template,
		appInfoFileName: appInfoFileName,
	}

	return jobFiles, nil

}

func getConfigMapLabelSelector(configMapName string) string {
	return fmt.Sprintf("%s=%s", FamilyNameLabelSelectorName, configMapName)
}

func getSmallestUnoccupiedIndex(configMaps []corev1.ConfigMap) int {
	occupationMap := make(map[string]bool)
	for _, configMap := range configMaps {
		occupationMap[configMap.Labels[FamilyIndexLabelSelectorName]] = true
	}

	for i := 1; i < len(configMaps); i++ {
		if !occupationMap[strconv.Itoa(i)] {
			return i
		}
	}

	return len(configMaps)
}

func getConfigMapName(name string, index int) string {
	if index == 0 {
		return name
	}
	return fmt.Sprintf("%s-%d", name, index)
}

func submitConfigMap(name, namespace string, generateName bool, clientset kubernetes.Interface) (*corev1.ConfigMap, error) {
	maybeConfigMapName := getConfigMapName(name, 0)

	configMap, err := createEmptyConfigMap(name, name, namespace, 0, clientset)
	if err == nil {
		return configMap, nil
	}

	if !generateName {
		return nil, fmt.Errorf("seems like there is another job with the name %s, you can use the --generate-name flag", maybeConfigMapName)
	}

	configMapLabelSelector := getConfigMapLabelSelector(maybeConfigMapName)
	for i := 0; i < ConfigMapGenerationRetries; i ++ {
		existingConfigMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{LabelSelector: configMapLabelSelector})
		if err != nil {
			return nil, err
		}
		configMapIndex := getSmallestUnoccupiedIndex(existingConfigMaps.Items)
		maybeConfigMapName = getConfigMapName(name, configMapIndex)

		configMap, err = createEmptyConfigMap(maybeConfigMapName, name, namespace, configMapIndex, clientset)
		if err == nil {
			return configMap, nil
		}
	}

	return nil, fmt.Errorf("could not create job, please try again later")
}

func createEmptyConfigMap(name, baseName, namespace string, index int, clientset kubernetes.Interface) (*corev1.ConfigMap, error) {
	labels := make(map[string]string)
	labels[kubectl.JOB_CONFIG_LABEL_KEY] = kubectl.JOB_CONFIG_LABEL_VALUES
	labels[FamilyIndexLabelSelectorName] = strconv.Itoa(index)
	labels[FamilyNameLabelSelectorName] = baseName

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	acceptedConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Create(&configMap)
	if err != nil {
		return nil, err
	}
	return acceptedConfigMap, nil
}

func populateConfigMap(configMap *corev1.ConfigMap, chartName, chartVersion, envValuesFile, valuesFileName, appInfoFileName, namespace string, clientset kubernetes.Interface) error {
	data := make(map[string]string)
	data[chartName] = chartVersion
	if envValuesFile != "" {
		envFileContent, err := ioutil.ReadFile(envValuesFile)
		if err != nil {
			return err
		}
		data["env-values"] = string(envFileContent)
	}
	valuesFileContent, err := ioutil.ReadFile(valuesFileName)
	if err != nil {
		return err
	}
	data["values"] = string(valuesFileContent)
	appFileContent, err := ioutil.ReadFile(appInfoFileName)
	if err != nil {
		return err
	}

	data["app"] = string(appFileContent)

	configMap.Data = data
	_, err = clientset.CoreV1().ConfigMaps(namespace).Update(configMap)
	return err
}

func cleanupJobFiles(files *JobFiles) {
	err := os.Remove(files.valueFileName)
	if err != nil {
		log.Warnf("Failed to delete %s due to %v", files.valueFileName, err)
	}
	err = os.Remove(files.template)
	if err != nil {
		log.Warnf("Failed to delete %s due to %v", files.valueFileName, err)
	}
	err = os.Remove(files.appInfoFileName)
	if err != nil {
		log.Warnf("Failed to delete %s due to %v", files.valueFileName, err)
	}
}

func submitJobInternal(name, namespace string, generateName bool, values interface{}, environmentValues string, chart string, clientset kubernetes.Interface) (string, error) {
	configMap, err := submitConfigMap(name, namespace, generateName, clientset)
	if err != nil {
		return "", err
	}
	jobName := configMap.Name
	jobFiles, err := generateJobFiles(jobName, namespace, values, environmentValues, chart)
	if err != nil {
		return jobName, err
	}
	defer cleanupJobFiles(jobFiles)

	chartName := helm.GetChartName(chart)
	chartVersion, err := helm.GetChartVersion(chart)
	if err != nil {
		return jobName, err
	}

	err = populateConfigMap(configMap, chartName, chartVersion, jobFiles.envValuesFile, jobFiles.valueFileName, jobFiles.appInfoFileName, namespace, clientset)
	if err != nil {
		return jobName, err
	}

	_, err = kubectl.InstallApps(jobFiles.template, namespace)
	if err != nil {
		return jobName, err
	}
	return jobName, nil
}

func SubmitJob(name, namespace string, generateName bool, values interface{}, environmentValues string, chart string, clientset kubernetes.Interface, dryRun bool) (string, error) {
	if dryRun {
		jobFiles, err := generateJobFiles(name, namespace, values, environmentValues, chart)
		if err != nil {
			return "", err
		}
		fmt.Println("Generate the template on:")
		fmt.Println(jobFiles.template)
		return "", nil
	}
	jobName, err := submitJobInternal(name, namespace, generateName, values, environmentValues, chart, clientset)
	if err != nil {
		return "", err
	}
	return jobName, nil
}