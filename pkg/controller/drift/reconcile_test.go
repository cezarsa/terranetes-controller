/*
 * Copyright (C) 2022  Appvia Ltd <info@appvia.io>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package drift

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1alpha1 "github.com/appvia/terranetes-controller/pkg/apis/core/v1alpha1"
	terraformv1alpha1 "github.com/appvia/terranetes-controller/pkg/apis/terraform/v1alpha1"
	"github.com/appvia/terranetes-controller/pkg/controller"
	"github.com/appvia/terranetes-controller/pkg/schema"
	controllertests "github.com/appvia/terranetes-controller/test"
	"github.com/appvia/terranetes-controller/test/fixtures"
)

func TestReconcile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Running Test Suite")
}

var _ = Describe("Drift Controller", func() {
	logrus.SetOutput(io.Discard)

	ctx := context.TODO()
	namespace := "default"

	When("configuration is reconciled", func() {
		cases := []struct {
			Name        string
			Before      func(ctrl *Controller)
			Check       func(configuration *terraformv1alpha1.Configuration)
			ShouldDrift bool
		}{
			{
				Name: "drift detection is not enabled",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					configuration.Spec.EnableDriftDetection = false
				},
			},
			/*
				{
					Name: "configuration is deleting",
					Check: func(configuration *terraformv1alpha1.Configuration) {
						configuration.Finalizers = []string{"do-not-delete"}
						cc.Update(ctx, configuration)
						cc.Delete(context.Background(), configuration)
						cc.Get(ctx, configuration.GetNamespacedName(), configuration)
					},
				},
			*/
			{
				Name: "terraform plan has not been run yet",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonNotDetermined
					cond.Status = metav1.ConditionFalse

					cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
					cond.Reason = corev1alpha1.ReasonNotDetermined
					cond.Status = metav1.ConditionFalse

				},
			},
			{
				Name: "terraform apply has not been run yet",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonComplete
					cond.Status = metav1.ConditionTrue

					cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
					cond.Reason = corev1alpha1.ReasonNotDetermined
					cond.Status = metav1.ConditionFalse
				},
			},
			{
				Name: "terraform plan has failed",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonError
					cond.Status = metav1.ConditionFalse
				},
			},
			{
				Name: "terraform apply has failed",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
					cond.Reason = corev1alpha1.ReasonError
					cond.Status = metav1.ConditionFalse
				},
			},
			{
				Name: "terraform plan in progress",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonInProgress
					cond.Status = metav1.ConditionFalse
				},
			},
			{
				Name: "terraform apply in progress",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
					cond.Reason = corev1alpha1.ReasonInProgress
					cond.Status = metav1.ConditionFalse
				},
			},
			{
				Name: "terraform plan occurred recently",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonComplete
					cond.LastTransitionTime = metav1.NewTime(time.Now())
					cond.Status = metav1.ConditionTrue
				},
			},
			{
				Name: "terraform apply occurred recently",
				Check: func(configuration *terraformv1alpha1.Configuration) {
					cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
					cond.Reason = corev1alpha1.ReasonComplete
					cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-24 * time.Hour))
					cond.Status = metav1.ConditionTrue

					cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
					cond.Reason = corev1alpha1.ReasonComplete
					cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Minute))
					cond.Status = metav1.ConditionTrue
				},
			},
			{
				Name: "we have multiple configuration in drift already",
				Before: func(ctrl *Controller) {
					for i := 0; i < 10; i++ {
						configuration := fixtures.NewValidBucketConfiguration(namespace, fmt.Sprintf("test%d-config", i))
						configuration.Annotations = map[string]string{terraformv1alpha1.DriftAnnotation: "true"}
						configuration.Spec.EnableDriftDetection = true

						controller.EnsureConditionsRegistered(terraformv1alpha1.DefaultConfigurationConditions, configuration)
						cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
						cond.Reason = corev1alpha1.ReasonInProgress
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						ctrl.cc.Create(ctx, configuration)
					}
				},
			},
			{
				Name: "we have a small drift threshold",
				Before: func(ctrl *Controller) {
					ctrl.DriftThreshold = 0.01

					for i := 0; i < 9; i++ {
						configuration := fixtures.NewValidBucketConfiguration(namespace, fmt.Sprintf("test%d-config", i))
						configuration.Annotations = map[string]string{terraformv1alpha1.DriftAnnotation: "true"}
						configuration.Spec.EnableDriftDetection = true

						controller.EnsureConditionsRegistered(terraformv1alpha1.DefaultConfigurationConditions, configuration)
						cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						ctrl.cc.Create(ctx, configuration)
					}
				},
				ShouldDrift: true,
			},
			{
				Name: "we have multiple configuration in different states",
				Before: func(ctrl *Controller) {
					// @step: we create x configuration not running
					for i := 0; i < 7; i++ {
						configuration := fixtures.NewValidBucketConfiguration(namespace, fmt.Sprintf("test-%d-notrunning", i))
						configuration.Annotations = map[string]string{terraformv1alpha1.DriftAnnotation: "true"}
						configuration.Spec.EnableDriftDetection = true

						controller.EnsureConditionsRegistered(terraformv1alpha1.DefaultConfigurationConditions, configuration)
						cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						ctrl.cc.Create(ctx, configuration)
					}

					// @step: we create x configuration running
					for i := 0; i < 2; i++ {
						configuration := fixtures.NewValidBucketConfiguration(namespace, fmt.Sprintf("test%d-running", i))
						configuration.Annotations = map[string]string{terraformv1alpha1.DriftAnnotation: "true"}
						configuration.Spec.EnableDriftDetection = true

						controller.EnsureConditionsRegistered(terraformv1alpha1.DefaultConfigurationConditions, configuration)
						cond := configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformPlan)
						cond.Reason = corev1alpha1.ReasonInProgress
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						cond = configuration.Status.GetCondition(terraformv1alpha1.ConditionTerraformApply)
						cond.Reason = corev1alpha1.ReasonComplete
						cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
						cond.ObservedGeneration = configuration.GetGeneration()
						cond.Status = metav1.ConditionTrue

						ctrl.cc.Create(ctx, configuration)
					}
				},
				ShouldDrift: false,
			},
			{
				Name:        "configuration should trigger a drift detection",
				ShouldDrift: true,
			},
		}

		for _, c := range cases {
			When(c.Name, func() {
				events := &controllertests.FakeRecorder{}
				ctrl := &Controller{
					CheckInterval:  5 * time.Minute,
					DriftInterval:  2 * time.Hour,
					DriftThreshold: 0.2,
					cc: fake.NewClientBuilder().
						WithScheme(schema.GetScheme()).
						WithStatusSubresource(&terraformv1alpha1.Configuration{}).
						Build(),
					recorder: events,
				}

				configuration := fixtures.NewValidBucketConfiguration(namespace, "test")
				configuration.Spec.EnableDriftDetection = true
				controller.EnsureConditionsRegistered(terraformv1alpha1.DefaultConfigurationConditions, configuration)

				// @step: set the conditions to true
				conditions := []corev1alpha1.ConditionType{
					terraformv1alpha1.ConditionTerraformPlan,
					terraformv1alpha1.ConditionTerraformApply,
				}
				for _, name := range conditions {
					cond := configuration.Status.GetCondition(name)
					cond.Reason = corev1alpha1.ReasonComplete
					cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
					cond.ObservedGeneration = configuration.GetGeneration()
					cond.Status = metav1.ConditionTrue
				}

				cond := configuration.Status.GetCondition(corev1alpha1.ConditionReady)
				cond.Reason = corev1alpha1.ReasonReady
				cond.LastTransitionTime = metav1.NewTime(time.Now().Add(-5 * time.Hour))
				cond.ObservedGeneration = configuration.GetGeneration()
				cond.Status = metav1.ConditionTrue

				if c.Before != nil {
					c.Before(ctrl)
				}
				if c.Check != nil {
					c.Check(configuration)
				}
				Expect(ctrl.cc.Create(ctx, configuration)).To(Succeed())

				It("should not return an error", func() {
					result, _, rerr := controllertests.Roll(ctx, ctrl, configuration, 1)

					Expect(rerr).To(BeNil())
					Expect(result.RequeueAfter).To(Equal(ctrl.CheckInterval))
				})

				switch c.ShouldDrift {
				case true:
					It("should have a drift detection annotation", func() {
						Expect(ctrl.cc.Get(ctx, configuration.GetNamespacedName(), configuration)).ToNot(HaveOccurred())
						Expect(configuration.GetAnnotations()).ToNot(BeEmpty())
						Expect(configuration.GetAnnotations()[terraformv1alpha1.DriftAnnotation]).ToNot(BeEmpty())
					})

					It("should have raised a event indicating the trigger", func() {
						Expect(events.Events).ToNot(BeEmpty())
						Expect(events.Events[0]).To(Equal("(default/test) Normal DriftDetection: Triggered drift detection on configuration"))
					})

				default:
					It("should not have a drift annotation", func() {
						Expect(ctrl.cc.Get(ctx, configuration.GetNamespacedName(), configuration)).ToNot(HaveOccurred())
						Expect(configuration.GetAnnotations()).To(BeEmpty())
					})
				}
			})
		}
	})
})
