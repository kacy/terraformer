// Copyright 2018 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/GoogleCloudPlatform/terraformer/terraform_utils"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/aws/aws-sdk-go-v2/aws"
)

var IamAllowEmptyValues = []string{"tags."}

var IamAdditionalFields = map[string]interface{}{}

type IamGenerator struct {
	AWSService
}

func (g *IamGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := iam.New(config)
	g.Resources = []terraform_utils.Resource{}
	err := g.getUsers(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getGroups(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getPolicies(svc)
	if err != nil {
		log.Println(err)
	}

	err = g.getRoles(svc)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func (g *IamGenerator) getRoles(svc *iam.Client) error {
	p := iam.NewListRolesPaginator(svc.ListRolesRequest(&iam.ListRolesInput{}))
	for p.Next(context.Background()) {
		for _, role := range p.CurrentPage().Roles {
			roleID := aws.StringValue(role.RoleId)
			roleName := aws.StringValue(role.RoleName)
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				roleID,
				roleName,
				"aws_iam_role",
				"aws",
				IamAllowEmptyValues))
			rolePoliciesPage := iam.NewListRolePoliciesPaginator(svc.ListRolePoliciesRequest(&iam.ListRolePoliciesInput{RoleName: role.RoleName}))
			for rolePoliciesPage.Next(context.Background()) {
				for _, policyName := range rolePoliciesPage.CurrentPage().PolicyNames {
					g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
						roleName+":"+policyName,
						roleName+"_"+policyName,
						"aws_iam_role_policy",
						"aws",
						IamAllowEmptyValues))
				}
			}
			if err := rolePoliciesPage.Err(); err != nil {
				log.Println(err)
				continue
			}
		}
	}
	return p.Err()
}

func (g *IamGenerator) getUsers(svc *iam.Client) error {
	p := iam.NewListUsersPaginator(svc.ListUsersRequest(&iam.ListUsersInput{}))
	for p.Next(context.Background()) {
		for _, user := range p.CurrentPage().Users {
			resourceName := aws.StringValue(user.UserName)
			g.Resources = append(g.Resources, terraform_utils.NewResource(
				resourceName,
				aws.StringValue(user.UserId),
				"aws_iam_user",
				"aws",
				map[string]string{
					"force_destroy": "false",
				},
				IamAllowEmptyValues,
				map[string]interface{}{}))
			err := g.getUserPolices(svc, user.UserName)
			if err != nil {
				log.Println(err)
			}
			//g.getUserGroup(svc, user.UserName) //not work maybe terraform-aws bug
		}
	}
	return p.Err()
}

func (g *IamGenerator) getUserGroup(svc *iam.Client, userName *string) error {
	p := iam.NewListGroupsForUserPaginator(svc.ListGroupsForUserRequest(&iam.ListGroupsForUserInput{UserName: userName}))
	for p.Next(context.Background()) {
		for _, group := range p.CurrentPage().Groups {
			resourceName := aws.StringValue(group.GroupName)
			groupIDAttachment := aws.StringValue(group.GroupName)
			g.Resources = append(g.Resources, terraform_utils.NewResource(
				groupIDAttachment,
				resourceName,
				"aws_iam_user_group_membership",
				"aws",
				map[string]string{"user": aws.StringValue(userName)},
				IamAllowEmptyValues,
				IamAdditionalFields,
			))
		}
	}
	return p.Err()
}

func (g *IamGenerator) getUserPolices(svc *iam.Client, userName *string) error {
	p := iam.NewListUserPoliciesPaginator(svc.ListUserPoliciesRequest(&iam.ListUserPoliciesInput{UserName: userName}))
	for p.Next(context.Background()) {
		for _, policy := range p.CurrentPage().PolicyNames {
			resourceName := aws.StringValue(userName) + "_" + policy
			resourceName = strings.Replace(resourceName, "@", "", -1)
			policyID := aws.StringValue(userName) + ":" + policy
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				policyID,
				resourceName,
				"aws_iam_user_policy",
				"aws",
				IamAllowEmptyValues))
		}
	}
	return p.Err()
}

func (g *IamGenerator) getPolicies(svc *iam.Client) error {
	p := iam.NewListPoliciesPaginator(svc.ListPoliciesRequest(&iam.ListPoliciesInput{Scope:iam.PolicyScopeTypeLocal}))
	for p.Next(context.Background()) {
		for _, policy := range p.CurrentPage().Policies {
			resourceName := aws.StringValue(policy.PolicyName)
			policyARN := aws.StringValue(policy.Arn)

			g.Resources = append(g.Resources, terraform_utils.NewResource(
				policyARN,
				resourceName,
				"aws_iam_policy_attachment",
				"aws",
				map[string]string{
					"policy_arn": policyARN,
					"name":       resourceName,
				},
				IamAllowEmptyValues,
				IamAdditionalFields))
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				policyARN,
				resourceName,
				"aws_iam_policy",
				"aws",
				IamAllowEmptyValues))

		}
	}
	return p.Err()
}

func (g *IamGenerator) getGroups(svc *iam.Client) error {
	p := iam.NewListGroupsPaginator(svc.ListGroupsRequest(&iam.ListGroupsInput{}))
	for p.Next(context.Background()) {
		for _, group := range p.CurrentPage().Groups {
			resourceName := aws.StringValue(group.GroupName)
			g.Resources = append(g.Resources, terraform_utils.NewSimpleResource(
				resourceName,
				resourceName,
				"aws_iam_group",
				"aws",
				IamAllowEmptyValues))
			g.Resources = append(g.Resources, terraform_utils.NewResource(
				resourceName,
				resourceName,
				"aws_iam_group_membership",
				"aws",
				map[string]string{
					"group": resourceName,
					"name":  resourceName,
				},
				[]string{"tags.", "users."},
				IamAdditionalFields))
			groupPoliciesPage := iam.NewListGroupPoliciesPaginator(svc.ListGroupPoliciesRequest(&iam.ListGroupPoliciesInput{GroupName: group.GroupName}))
			for groupPoliciesPage.Next(context.Background()) {
				for _, policy := range groupPoliciesPage.CurrentPage().PolicyNames {
					id := resourceName + ":" + policy
					groupPolicyName := resourceName + "_" + policy
					g.Resources = append(g.Resources, terraform_utils.NewResource(
						id,
						groupPolicyName,
						"aws_iam_group_policy",
						"aws",
						map[string]string{},
						IamAllowEmptyValues,
						IamAdditionalFields))
				}
			}
			if err := groupPoliciesPage.Err(); err != nil {
				log.Println(err)
			}
		}
	}
	return p.Err()
}

// PostGenerateHook for add policy json as heredoc
func (g *IamGenerator) PostConvertHook() error {
	for i, resource := range g.Resources {
		if resource.InstanceInfo.Type == "aws_iam_policy" ||
			resource.InstanceInfo.Type == "aws_iam_user_policy" ||
			resource.InstanceInfo.Type == "aws_iam_group_policy" ||
			resource.InstanceInfo.Type == "aws_iam_role_policy" {
			policy := resource.Item["policy"].(string)
			resource.Item["policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
		} else if resource.InstanceInfo.Type == "aws_iam_role" {
			policy := resource.Item["assume_role_policy"].(string)
			g.Resources[i].Item["assume_role_policy"] = fmt.Sprintf(`<<POLICY
%s
POLICY`, policy)
		}
	}
	return nil
}
