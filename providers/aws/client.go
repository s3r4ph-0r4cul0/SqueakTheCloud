package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/squeak-the-cloud/squeak/output"
)

type PolicyAnalysis struct {
	PolicyType             string   `json:"policy_type"` // "inline" or "attached"
	Name                   string   `json:"name"`
	HasFullAdmin           bool     `json:"has_full_admin"` // Allow Action * on Resource *
	WildcardActions        []string `json:"wildcard_actions"`
	WildcardResources      []string `json:"wildcard_resources"`
	PrivilegeEscalationPaths []string `json:"privilege_escalation_paths"`
}

type RoleAuditResult struct {
	RoleName                 string            `json:"role_name"`
	RoleArn                  string            `json:"role_arn"`
	AssumeRolePolicyDocument string            `json:"assume_role_policy_document"`
	AttachedPolicies         []string          `json:"attached_policies"`
	InlinePolicies           []string          `json:"inline_policies"`
	PolicyAnalyses           []PolicyAnalysis  `json:"policy_analyses"`
	Tags                     map[string]string `json:"tags"`
	NonAuditablePolicies     []string          `json:"non_auditable_policies"`
}

type IdentityProviderAuditResult struct {
	ProviderArn string            `json:"provider_arn"`
	Type        string            `json:"type"` // "SAML" or "OIDC"
	Url         string            `json:"url,omitempty"`
	Tags        map[string]string `json:"tags"`
}

type UserAuditResult struct {
	UserName             string            `json:"user_name"`
	UserArn              string            `json:"user_arn"`
	MfaEnabled           bool              `json:"mfa_enabled"`
	KeysCount            int               `json:"keys_count"`
	HasOldKeys           bool              `json:"has_old_keys"` // keys > 90 days
	KeysDetails          []string          `json:"keys_details"`
	PasswordLastUsed     string            `json:"password_last_used,omitempty"`
	NonAuditableDetails  []string          `json:"non_auditable_details"`
	Tags                 map[string]string `json:"tags"`
}

type AccountSecurityAuditResult struct {
	RootMfaEnabled              bool     `json:"root_mfa_enabled"`
	RootAccessKeysPresent       bool     `json:"root_access_keys_present"`
	PasswordPolicyExists        bool     `json:"password_policy_exists"`
	PasswordMinLength           int32    `json:"password_min_length,omitempty"`
	PasswordRequireUppercase    bool     `json:"password_require_uppercase"`
	PasswordRequireLowercase    bool     `json:"password_require_lowercase"`
	PasswordRequireNumbers      bool     `json:"password_require_numbers"`
	PasswordRequireSymbols      bool     `json:"password_require_symbols"`
	PasswordExpireDays          int32    `json:"password_expire_days,omitempty"`
	NonAuditableDetails         []string `json:"non_auditable_details"`
}

type TrailAudit struct {
	Name                string `json:"name"`
	HomeRegion          string `json:"home_region"`
	IsMultiRegionTrail  bool   `json:"is_multi_region_trail"`
	IsLogging           bool   `json:"is_logging"`
	LatestDeliveryTime  string `json:"latest_delivery_time,omitempty"`
}

type LoggingAuditResult struct {
	Trails              []TrailAudit `json:"trails"`
	NonAuditableDetails []string     `json:"non_auditable_details"`
}

func Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output.LogInfo("Initializing AWS configuration...")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to load AWS config: %v", err))
		return
	}

	// Dynamic AWS Account ID discovery using STS
	stsClient := sts.NewFromConfig(cfg)
	callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		output.LogWarning(fmt.Sprintf("could not discover dynamic AWS account context via STS: %v", err))
	} else {
		output.LogSuccess(fmt.Sprintf("Auditing AWS Account ID: %s", aws.ToString(callerIdentity.Account)))
	}

	iamClient := iam.NewFromConfig(cfg)

	// 1. Audit IAM Roles with Pagination Loop
	output.LogInfo("Auditing IAM Roles...")
	var roles []iamTypes.Role
	var marker *string

	for {
		rolesOutput, err := iamClient.ListRoles(ctx, &iam.ListRolesInput{
			Marker: marker,
		})
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to list IAM roles: %v", err))
			break
		}

		roles = append(roles, rolesOutput.Roles...)
		if rolesOutput.Marker == nil || *rolesOutput.Marker == "" {
			break
		}
		marker = rolesOutput.Marker
	}

	for index, role := range roles {
		roleName := aws.ToString(role.RoleName)
		output.LogInfo(fmt.Sprintf("Auditing role: %s", roleName))

		result := RoleAuditResult{
			RoleName:                 roleName,
			RoleArn:                  aws.ToString(role.Arn),
			AssumeRolePolicyDocument: aws.ToString(role.AssumeRolePolicyDocument),
			AttachedPolicies:         []string{},
			InlinePolicies:           []string{},
			PolicyAnalyses:           []PolicyAnalysis{},
			Tags:                     make(map[string]string),
			NonAuditablePolicies:     []string{},
		}

		// Get Attached Policies
		attached, err := iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: role.RoleName,
		})
		if err == nil {
			for _, p := range attached.AttachedPolicies {
				policyName := aws.ToString(p.PolicyName)
				result.AttachedPolicies = append(result.AttachedPolicies, policyName)

				// Fetch Managed Policy Document for Static Analysis
				policyInfo, err := iamClient.GetPolicy(ctx, &iam.GetPolicyInput{
					PolicyArn: p.PolicyArn,
				})
				if err != nil {
					result.NonAuditablePolicies = append(result.NonAuditablePolicies, fmt.Sprintf("attached:%s (GetPolicy: %v)", policyName, err))
					continue
				}

				if policyInfo.Policy != nil {
					versionInfo, err := iamClient.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
						PolicyArn: p.PolicyArn,
						VersionId: policyInfo.Policy.DefaultVersionId,
					})
					if err != nil {
						result.NonAuditablePolicies = append(result.NonAuditablePolicies, fmt.Sprintf("attached:%s (GetPolicyVersion: %v)", policyName, err))
						continue
					}

					if versionInfo.PolicyVersion != nil {
						analysis := analyzePolicyDocument(policyName, "attached", aws.ToString(versionInfo.PolicyVersion.Document))
						result.PolicyAnalyses = append(result.PolicyAnalyses, analysis)
					}
				}
			}
		} else {
			result.NonAuditablePolicies = append(result.NonAuditablePolicies, fmt.Sprintf("list_attached_policies_failed: %v", err))
		}

		// Get Inline Policies
		inline, err := iamClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
			RoleName: role.RoleName,
		})
		if err == nil {
			for _, pName := range inline.PolicyNames {
				result.InlinePolicies = append(result.InlinePolicies, pName)

				// Fetch Inline Policy Document for Static Analysis
				inlineDoc, err := iamClient.GetRolePolicy(ctx, &iam.GetRolePolicyInput{
					RoleName:   role.RoleName,
					PolicyName: aws.String(pName),
				})
				if err != nil {
					result.NonAuditablePolicies = append(result.NonAuditablePolicies, fmt.Sprintf("inline:%s (GetRolePolicy: %v)", pName, err))
					continue
				}

				analysis := analyzePolicyDocument(pName, "inline", aws.ToString(inlineDoc.PolicyDocument))
				result.PolicyAnalyses = append(result.PolicyAnalyses, analysis)
			}
		} else {
			result.NonAuditablePolicies = append(result.NonAuditablePolicies, fmt.Sprintf("list_inline_policies_failed: %v", err))
		}

		// Get Tags
		tags, err := iamClient.ListRoleTags(ctx, &iam.ListRoleTagsInput{
			RoleName: role.RoleName,
		})
		if err == nil {
			for _, t := range tags.Tags {
				result.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}
		}

		filename := fmt.Sprintf("aws_iam_role_%d.json", index+1)
		if writeErr := output.WriteResult(filename, result); writeErr != nil {
			output.LogCritical(fmt.Sprintf("failed to write audit result for role %s: %v", roleName, writeErr))
		} else {
			output.LogSuccess(fmt.Sprintf("role %s audited", roleName))
		}
	}

	// 2. Audit OIDC Providers
	output.LogInfo("Auditing OIDC Providers...")
	oidcOutput, err := iamClient.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to list OIDC providers: %v", err))
	} else {
		for index, provider := range oidcOutput.OpenIDConnectProviderList {
			arn := aws.ToString(provider.Arn)
			output.LogInfo(fmt.Sprintf("Auditing OIDC provider: %s", arn))

			result := IdentityProviderAuditResult{
				ProviderArn: arn,
				Type:        "OIDC",
				Tags:        make(map[string]string),
			}

			details, err := iamClient.GetOpenIDConnectProvider(ctx, &iam.GetOpenIDConnectProviderInput{
				OpenIDConnectProviderArn: provider.Arn,
			})
			if err == nil {
				result.Url = aws.ToString(details.Url)
				for _, t := range details.Tags {
					result.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
				}
			}

			filename := fmt.Sprintf("aws_oidc_provider_%d.json", index+1)
			if writeErr := output.WriteResult(filename, result); writeErr != nil {
				output.LogCritical(fmt.Sprintf("failed to write OIDC provider audit: %v", writeErr))
			}
		}
	}

	// 3. Audit SAML Providers
	output.LogInfo("Auditing SAML Providers...")
	samlOutput, err := iamClient.ListSAMLProviders(ctx, &iam.ListSAMLProvidersInput{})
	if err != nil {
		output.LogCritical(fmt.Sprintf("failed to list SAML providers: %v", err))
	} else {
		for index, provider := range samlOutput.SAMLProviderList {
			arn := aws.ToString(provider.Arn)
			output.LogInfo(fmt.Sprintf("Auditing SAML provider: %s", arn))

			result := IdentityProviderAuditResult{
				ProviderArn: arn,
				Type:        "SAML",
				Tags:        make(map[string]string),
			}

			details, err := iamClient.GetSAMLProvider(ctx, &iam.GetSAMLProviderInput{
				SAMLProviderArn: provider.Arn,
			})
			if err == nil {
				for _, t := range details.Tags {
					result.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
				}
			}

			filename := fmt.Sprintf("aws_saml_provider_%d.json", index+1)
			if writeErr := output.WriteResult(filename, result); writeErr != nil {
				output.LogCritical(fmt.Sprintf("failed to write SAML provider audit: %v", writeErr))
			}
		}
	}

	// 4. Audit IAM Users (New capability)
	output.LogInfo("Auditing IAM Users...")
	var users []iamTypes.User
	var userMarker *string
	for {
		usersOutput, err := iamClient.ListUsers(ctx, &iam.ListUsersInput{
			Marker: userMarker,
		})
		if err != nil {
			output.LogCritical(fmt.Sprintf("failed to list IAM users: %v", err))
			break
		}
		users = append(users, usersOutput.Users...)
		if usersOutput.Marker == nil || *usersOutput.Marker == "" {
			break
		}
		userMarker = usersOutput.Marker
	}

	for index, user := range users {
		userName := aws.ToString(user.UserName)
		output.LogInfo(fmt.Sprintf("Auditing user: %s", userName))

		userResult := UserAuditResult{
			UserName:            userName,
			UserArn:             aws.ToString(user.Arn),
			MfaEnabled:          false,
			KeysCount:           0,
			HasOldKeys:          false,
			KeysDetails:         []string{},
			NonAuditableDetails: []string{},
			Tags:                make(map[string]string),
		}

		if user.PasswordLastUsed != nil {
			userResult.PasswordLastUsed = user.PasswordLastUsed.Format(time.RFC3339)
		}

		// Check MFA
		mfa, err := iamClient.ListMFADevices(ctx, &iam.ListMFADevicesInput{
			UserName: user.UserName,
		})
		if err == nil {
			userResult.MfaEnabled = len(mfa.MFADevices) > 0
		} else {
			userResult.NonAuditableDetails = append(userResult.NonAuditableDetails, fmt.Sprintf("mfa_check_failed: %v", err))
		}

		// Check Access Keys
		keys, err := iamClient.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
			UserName: user.UserName,
		})
		if err == nil {
			userResult.KeysCount = len(keys.AccessKeyMetadata)
			now := time.Now()
			for _, k := range keys.AccessKeyMetadata {
				created := aws.ToTime(k.CreateDate)
				age := now.Sub(created)
				keyInfo := fmt.Sprintf("key_id:%s,status:%s,created:%s", aws.ToString(k.AccessKeyId), string(k.Status), created.Format(time.RFC3339))
				userResult.KeysDetails = append(userResult.KeysDetails, keyInfo)

				if age.Hours() > 24*90 { // older than 90 days
					userResult.HasOldKeys = true
				}
			}
		} else {
			userResult.NonAuditableDetails = append(userResult.NonAuditableDetails, fmt.Sprintf("keys_check_failed: %v", err))
		}

		// Get Tags
		userTags, err := iamClient.ListUserTags(ctx, &iam.ListUserTagsInput{
			UserName: user.UserName,
		})
		if err == nil {
			for _, t := range userTags.Tags {
				userResult.Tags[aws.ToString(t.Key)] = aws.ToString(t.Value)
			}
		}

		filename := fmt.Sprintf("aws_iam_user_%d.json", index+1)
		if writeErr := output.WriteResult(filename, userResult); writeErr != nil {
			output.LogCritical(fmt.Sprintf("failed to write user audit result: %v", writeErr))
		}
	}

	// 5. Audit Account Security settings (MFA Root, Password Policy) (New capability)
	output.LogInfo("Auditing Account Security Status...")
	accountResult := AccountSecurityAuditResult{
		RootMfaEnabled:        false,
		RootAccessKeysPresent: false,
		PasswordPolicyExists:  false,
		NonAuditableDetails:   []string{},
	}

	summary, err := iamClient.GetAccountSummary(ctx, &iam.GetAccountSummaryInput{})
	if err == nil {
		if mfaVal, ok := summary.SummaryMap["AccountMFAEnabled"]; ok {
			accountResult.RootMfaEnabled = mfaVal > 0
		}
		if keysVal, ok := summary.SummaryMap["AccountAccessKeysPresent"]; ok {
			accountResult.RootAccessKeysPresent = keysVal > 0
		}
	} else {
		accountResult.NonAuditableDetails = append(accountResult.NonAuditableDetails, fmt.Sprintf("get_account_summary_failed: %v", err))
	}

	passPolicy, err := iamClient.GetAccountPasswordPolicy(ctx, &iam.GetAccountPasswordPolicyInput{})
	if err == nil {
		accountResult.PasswordPolicyExists = true
		if passPolicy.PasswordPolicy != nil {
			accountResult.PasswordMinLength = aws.ToInt32(passPolicy.PasswordPolicy.MinimumPasswordLength)
			accountResult.PasswordRequireUppercase = passPolicy.PasswordPolicy.RequireUppercaseCharacters
			accountResult.PasswordRequireLowercase = passPolicy.PasswordPolicy.RequireLowercaseCharacters
			accountResult.PasswordRequireNumbers = passPolicy.PasswordPolicy.RequireNumbers
			accountResult.PasswordRequireSymbols = passPolicy.PasswordPolicy.RequireSymbols
			if passPolicy.PasswordPolicy.MaxPasswordAge != nil {
				accountResult.PasswordExpireDays = aws.ToInt32(passPolicy.PasswordPolicy.MaxPasswordAge)
			}
		}
	} else {
		// GetAccountPasswordPolicy returns a NoSuchEntity error if no custom password policy exists.
		if strings.Contains(err.Error(), "NoSuchEntity") {
			accountResult.PasswordPolicyExists = false
		} else {
			accountResult.NonAuditableDetails = append(accountResult.NonAuditableDetails, fmt.Sprintf("get_password_policy_failed: %v", err))
		}
	}

	if writeErr := output.WriteResult("aws_account_security.json", accountResult); writeErr != nil {
		output.LogCritical(fmt.Sprintf("failed to write AWS account security audit: %v", writeErr))
	}

	// 6. Audit CloudTrail Logs (New capability)
	output.LogInfo("Auditing CloudTrail Logs...")
	ctResult := LoggingAuditResult{
		Trails:              []TrailAudit{},
		NonAuditableDetails: []string{},
	}

	ctClient := cloudtrail.NewFromConfig(cfg)
	trails, err := ctClient.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{
		TrailNameList: []string{},
	})
	if err == nil {
		for _, t := range trails.TrailList {
			status, err := ctClient.GetTrailStatus(ctx, &cloudtrail.GetTrailStatusInput{
				Name: t.Name,
			})
			isLogging := false
			latestDelivery := ""
			if err == nil {
				isLogging = aws.ToBool(status.IsLogging)
				if status.LatestDeliveryTime != nil {
					latestDelivery = status.LatestDeliveryTime.Format(time.RFC3339)
				}
			} else {
				ctResult.NonAuditableDetails = append(ctResult.NonAuditableDetails, fmt.Sprintf("trail:%s_status_failed: %v", aws.ToString(t.Name), err))
			}

			ctResult.Trails = append(ctResult.Trails, TrailAudit{
				Name:               aws.ToString(t.Name),
				HomeRegion:         aws.ToString(t.HomeRegion),
				IsMultiRegionTrail: aws.ToBool(t.IsMultiRegionTrail),
				IsLogging:          isLogging,
				LatestDeliveryTime: latestDelivery,
			})
		}
	} else {
		ctResult.NonAuditableDetails = append(ctResult.NonAuditableDetails, fmt.Sprintf("describe_trails_failed: %v", err))
	}

	if writeErr := output.WriteResult("aws_logging_security.json", ctResult); writeErr != nil {
		output.LogCritical(fmt.Sprintf("failed to write AWS logging security audit: %v", writeErr))
	}
}

func analyzePolicyDocument(name string, policyType string, rawDoc string) PolicyAnalysis {
	analysis := PolicyAnalysis{
		PolicyType:               policyType,
		Name:                     name,
		PrivilegeEscalationPaths: []string{},
	}

	decoded, err := url.QueryUnescape(rawDoc)
	if err != nil {
		return analysis
	}

	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &doc); err != nil {
		return analysis
	}

	statementsRaw, exists := doc["Statement"]
	if !exists {
		return analysis
	}

	var statements []interface{}
	switch v := statementsRaw.(type) {
	case []interface{}:
		statements = v
	case map[string]interface{}:
		statements = append(statements, v)
	}

	// Critical privilege escalation APIs to match
	escalationAPIs := map[string]string{
		"iam:createaccesskey":        "CreateAccessKey (create a key for another user to compromise them)",
		"iam:createloginprofile":     "CreateLoginProfile (set a password for a console user)",
		"iam:updateloginprofile":     "UpdateLoginProfile (change a password for a console user)",
		"iam:attachuserpolicy":       "AttachUserPolicy (attach admin policy to user)",
		"iam:attachrolepolicy":       "AttachRolePolicy (attach admin policy to role)",
		"iam:attachgrouppolicy":      "AttachGroupPolicy (attach admin policy to group)",
		"iam:putuserpolicy":          "PutUserPolicy (write admin inline policy to user)",
		"iam:putrolepolicy":          "PutRolePolicy (write admin inline policy to role)",
		"iam:putgrouppolicy":         "PutGroupPolicy (write admin inline policy to group)",
		"iam:addusertogroup":         "AddUserToGroup (add oneself to an admin group)",
		"iam:passrole":               "PassRole (pass privilege role to a service to launch admin tasks)",
		"lambda:createfunction":      "CreateFunction (create a lambda with high-privilege role)",
		"lambda:updatefunctioncode":  "UpdateFunctionCode (inject backdoor code into high-privilege lambda)",
		"iam:createpolicyversion":    "CreatePolicyVersion (modify policy defaults to gain admin access)",
	}

	for _, stmtRaw := range statements {
		stmt, ok := stmtRaw.(map[string]interface{})
		if !ok {
			continue
		}

		effect, _ := stmt["Effect"].(string)
		if effect != "Allow" {
			continue
		}

		var actions []string
		if actRaw, ok := stmt["Action"]; ok {
			switch act := actRaw.(type) {
			case string:
				actions = append(actions, act)
			case []interface{}:
				for _, a := range act {
					if s, ok := a.(string); ok {
						actions = append(actions, s)
					}
				}
			}
		}

		var resources []string
		if resRaw, ok := stmt["Resource"]; ok {
			switch res := resRaw.(type) {
			case string:
				resources = append(resources, res)
			case []interface{}:
				for _, r := range res {
					if s, ok := r.(string); ok {
						resources = append(resources, s)
					}
				}
			}
		}

		hasStarAction := false
		hasStarResource := false

		for _, action := range actions {
			lowerAct := strings.ToLower(action)
			if lowerAct == "*" {
				hasStarAction = true
				analysis.WildcardActions = append(analysis.WildcardActions, action)
				// Full wildcards encompass all escalation pathways
				for _, pathDesc := range escalationAPIs {
					alreadyPresent := false
					for _, existing := range analysis.PrivilegeEscalationPaths {
						if existing == pathDesc {
							alreadyPresent = true
							break
						}
					}
					if !alreadyPresent {
						analysis.PrivilegeEscalationPaths = append(analysis.PrivilegeEscalationPaths, pathDesc)
					}
				}
			} else if len(action) > 1 && action[len(action)-1] == '*' {
				analysis.WildcardActions = append(analysis.WildcardActions, action)
				// Check prefix match for escalation APIs
				prefix := strings.TrimSuffix(lowerAct, "*")
				for key, pathDesc := range escalationAPIs {
					if strings.HasPrefix(key, prefix) {
						alreadyPresent := false
						for _, existing := range analysis.PrivilegeEscalationPaths {
							if existing == pathDesc {
								alreadyPresent = true
								break
							}
						}
						if !alreadyPresent {
							analysis.PrivilegeEscalationPaths = append(analysis.PrivilegeEscalationPaths, pathDesc)
						}
					}
				}
			} else {
				// Exact match
				if pathDesc, exists := escalationAPIs[lowerAct]; exists {
					alreadyPresent := false
					for _, existing := range analysis.PrivilegeEscalationPaths {
						if existing == pathDesc {
							alreadyPresent = true
							break
						}
					}
					if !alreadyPresent {
						analysis.PrivilegeEscalationPaths = append(analysis.PrivilegeEscalationPaths, pathDesc)
					}
				}
			}
		}

		for _, resource := range resources {
			if resource == "*" {
				hasStarResource = true
				analysis.WildcardResources = append(analysis.WildcardResources, resource)
			}
		}

		if hasStarAction && hasStarResource {
			analysis.HasFullAdmin = true
		}
	}

	return analysis
}
