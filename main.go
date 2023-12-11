package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

const (
	tfDir            = "terraform"
	tfLocalsFileName = "locals.tf"
	tfMainFileName   = "main.tf"
)

func makeDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModeDir|0755)
	} else if err != nil {
		panic(err)
	}
}

func createFile(fileName string) *os.File {
	file, err := os.Create(tfDir + "/" + fileName)
	if err != nil {
		panic(err)
	}
	return file
}

func createTerraformLocals() {
	hclFile := hclwrite.NewEmptyFile()
	makeDir(tfDir)
	tfFile := createFile(tfLocalsFileName)

	terraformLocals := hclFile.Body().AppendNewBlock("locals", nil)

	// NOTE: Instead of using random values
	// we implement functions to call IBM Cloud Secret Manager API
	// to build real secrets field and CRN values

	secretsMap := make(map[string]cty.Value)
	indexSecret := 0
	for indexSecret < 5 {
		indexField := 0
		fieldsMap := make(map[string]cty.Value)
		crnMap := make(map[string]cty.Value)
		for indexField < 5 {
			crnMap["field-"+fmt.Sprint(fmt.Sprint(rand.Int()))] = cty.StringVal("crn:v1:bluemix:" + fmt.Sprint(rand.Int()))
			indexField++
		}
		fieldsMap["fields"] = cty.ObjectVal(crnMap)
		secretsMap["secret-"+fmt.Sprint(indexSecret)] = cty.ObjectVal(fieldsMap)
		indexSecret++
	}

	terraformLocals.Body().SetAttributeValue("secrets", cty.ObjectVal(secretsMap))
	tfFile.Write(hclFile.Bytes())
}

func createTerraformMain() {
	hclFile := hclwrite.NewEmptyFile()
	makeDir(tfDir)
	tfFile := createFile(tfMainFileName)

	resource := hclFile.Body().AppendNewBlock("resource", []string{"ibm_container_ingress_secret_opaque", "ingress-secret"})
	resourceBody := resource.Body()

	resourceBody.SetAttributeTraversal("for_each", hcl.Traversal{
		hcl.TraverseRoot{Name: "local"},
		hcl.TraverseAttr{Name: "secrets"},
	})
	resourceBody.SetAttributeValue("cluster", cty.StringVal(os.Getenv("CLUSTER_ID")))
	resourceBody.SetAttributeTraversal("secret_name", hcl.Traversal{
		hcl.TraverseRoot{Name: "each"},
		hcl.TraverseAttr{Name: "key"},
	})
	resourceBody.SetAttributeValue("secret_namespace", cty.StringVal(os.Getenv("NAMESPACE")))

	dynamicBlock := resourceBody.AppendNewBlock("dynamic", []string{"fields"})
	dynamicBlockBody := dynamicBlock.Body()

	dynamicBlockBody.SetAttributeTraversal("for_each", hcl.Traversal{
		hcl.TraverseRoot{Name: "local.secrets[each.key]"},
		hcl.TraverseAttr{Name: "fields"},
	})

	contentBlock := dynamicBlockBody.AppendNewBlock("content", nil)
	contentBlockBody := contentBlock.Body()

	contentBlockBody.SetAttributeTraversal("field_name", hcl.Traversal{
		hcl.TraverseRoot{Name: "fields"},
		hcl.TraverseAttr{Name: "key"},
	})
	contentBlockBody.SetAttributeTraversal("crn", hcl.Traversal{
		hcl.TraverseRoot{Name: "fields"},
		hcl.TraverseAttr{Name: "value"},
	})

	tfFile.Write(hclFile.Bytes())
}

func main() {
	createTerraformLocals()
	createTerraformMain()
}
